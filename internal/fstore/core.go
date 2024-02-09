package fstore

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

const (
	FileIdPrefix = "file_" // prefix of file_id

	StatusNormal    = "NORMAL"  // file.status - normal
	StatusLogicDel  = "LOG_DEL" // file.status - logically deletedy
	StatusPhysicDel = "PHY_DEL" // file.status - physically deletedy

	PdelStrategyDirect = "direct" // file delete strategy - direct
	PdelStrategyTrash  = "trash"  // file delete strategy - trash

	ByteRangeMaxSize = 30_000_000 // 30 mb
)

var (
	ErrFileNotFound = miso.NewErrf("File is not found").WithCode(api.FileNotFound)
	ErrFileDeleted  = miso.NewErrf("File has been deleted already").WithCode(api.FileDeleted)

	fileIdExistCache = miso.NewRCache[string]("fstore:fileid:exist:v1:",
		miso.RCacheConfig{
			Exp:    10 * time.Minute,
			NoSync: true,
		},
	)
)

func init() {
	miso.SetDefProp(PropStorageDir, "./storage")
	miso.SetDefProp(PropTrashDir, "./trash")
	miso.SetDefProp(PropPDelStrategy, PdelStrategyTrash)
	miso.SetDefProp(PropSanitizeStorageTaskDryRun, false)
	miso.SetDefProp(PropTempDir, "/tmp")
}

type ByteRange struct {
	zero  bool  // whether the byte range is not specified (so called, zero value)
	Start int64 // start of byte range (inclusive)
	End   int64 // end of byte range (inclusive)
}

func (br ByteRange) Size() int64 {
	if br.IsZero() {
		return 0
	}
	return br.End - br.Start + 1
}

func (br ByteRange) IsZero() bool {
	return br.zero
}

func ZeroByteRange() ByteRange {
	return ByteRange{true, -1, -1}
}

type CachedFile struct {
	FileId string `json:"fileId"`
	Name   string `json:"name"`
}

type PDelFileOp interface {
	/*
		Delete file for the given fileId.

		Implmentation should detect whether the file still exists before undertaking deletion.
		If file has been deleted, nil error should be returned
	*/
	delete(r miso.Rail, fileId string) error
}

// The 'direct' implementation of of PDelFileOp, files are deleted directly
type PDelFileDirectOp struct {
}

func (p PDelFileDirectOp) delete(rail miso.Rail, fileId string) error {
	// symbolink file is of course not found, attempting to remove it is harmless
	file := GenStoragePath(fileId)
	er := os.Remove(file)
	if er != nil {
		if os.IsNotExist(er) {
			rail.Infof("File has been deleted, file: %s", file)
			return nil
		}

		rail.Errorf("Failed to delete file, file: %s, %v", file, er)
		return er
	}

	return nil
}

// The 'trash' implementation of of PDelFileOp, files are deleted directly
type PDelFileTrashOp struct {
}

func (p PDelFileTrashOp) delete(rail miso.Rail, fileId string) error {
	// symbolink file is of course not found, attempting to move it is harmless
	frm := GenStoragePath(fileId)
	to := GenTrashPath(fileId)

	if e := os.Rename(frm, to); e != nil {
		if os.IsNotExist(e) {
			rail.Infof("File has been deleted, file: %s", frm)
			return nil
		}
		return fmt.Errorf("failed to rename file from %s, to %s, %v", frm, to, e)
	}

	rail.Infof("Renamed file from %s, to %s", frm, to)
	return nil
}

type File struct {
	Id         int64       `json:"id"`
	FileId     string      `json:"fileId"`
	Link       string      `json:"-"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	Size       int64       `json:"size"`
	Md5        string      `json:"md5"`
	UplTime    miso.ETime  `json:"uplTime"`
	LogDelTime *miso.ETime `json:"logDelTime"`
	PhyDelTime *miso.ETime `json:"phyDelTime"`
}

// Check whether current file is of zero value
func (f *File) IsZero() bool {
	return f.Id <= 0
}

// Check if the file is deleted already
func (f *File) IsDeleted() bool {
	return f.Status != StatusNormal
}

// Check if the file is logically already
func (f *File) IsLogiDeleted() bool {
	return f.Status == StatusLogicDel
}

// Return the actual storage path including symbolic link.
//
// A file could be a symbolic link to another file (using field f.Link).
//
// Be cautious if this path is used to delete/remove files (i.e., it shouldn't).
func (f *File) StoragePath() string {
	dfileId := f.FileId
	if f.Link != "" {
		dfileId = f.Link
	}
	return GenStoragePath(dfileId)
}

// Generate random file_id
func GenFileId() string {
	return miso.GenIdP(FileIdPrefix)
}

// Initialize storage dir
//
// Property `fstore.storage.dir` is used
func InitStorageDir(rail miso.Rail) error {
	dir := miso.GetPropStr(PropStorageDir)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to MkdirAll, %v", err)
	}
	return nil
}

// Initialize trash dir
//
// Property `fstore.trash.dir` is used
func InitTrashDir(rail miso.Rail) error {
	dir := miso.GetPropStr(PropTrashDir)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to MkdirAll, %v", err)
	}
	return nil
}

// Generate file path
//
// Property `fstore.storage.dir` is used
func GenStoragePath(fileId string) string {
	dir := miso.GetPropStr(PropStorageDir)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir + fileId
}

// Generate file path for trashed file
//
// Property `fstore.trash.dir` is used
func GenTrashPath(fileId string) string {
	dir := miso.GetPropStr(PropTrashDir)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir + fileId
}

/*
List logically deleted files, and based on the configured strategy, deleted them 'physically'.

This func reads property 'fstore.pdelete.strategy'.

If strategy is 'direct', files are deleted directly. If strategy is 'trash' (default),
files are moved to 'trash' directory, which is specified in property 'fstore.trash.dir'

This func should only be used during server maintenance (no one can upload file).
*/
func RemoveDeletedFiles(rail miso.Rail, db *gorm.DB) error {
	start := time.Now()
	defer miso.TimeOp(rail, start, "BatchPhyDelFiles")

	before := start.Add(-1 * time.Hour) // only delete files that are logically deleted 1 hour ago
	var minId int = 0
	var l []PendingPhyDelFile
	var err error
	strat := miso.GetPropStr(PropPDelStrategy)
	delFileOp := NewPDelFileOp(strat)

	for {
		if l, err = listPendingPhyDelFiles(rail, db, before, minId); err != nil {
			return fmt.Errorf("failed to listPendingPhyDelFiles, %v", err)
		}
		if len(l) < 1 {
			return nil
		}

		for _, f := range l {
			if e := PhyDelFile(rail, db, f.FileId, delFileOp); e != nil {
				rail.Errorf("Failed to PhyDelFile, strategy: %v, fileId: %s, %v", strat, f.FileId, e)
			}
		}
		minId = l[len(l)-1].Id
		rail.Debugf("BatchPhyDelFiles, minId: %v", minId)
	}
}

type PendingPhyDelFile struct {
	Id     int
	FileId string
}

func listPendingPhyDelFiles(rail miso.Rail, db *gorm.DB, beforeLogDelTime time.Time, minId int) ([]PendingPhyDelFile, error) {
	defer miso.TimeOp(rail, time.Now(), "listPendingPhyDelFiles")

	var l []PendingPhyDelFile
	tx := db.Raw("select id, file_id from file where id > ? and status = ? and log_del_time <= ? order by id asc limit 500",
		minId, StatusLogicDel, beforeLogDelTime).
		Scan(&l)

	if e := tx.Error; e != nil {
		rail.Errorf("Failed to list LDel files, %v", e)
		return nil, e
	}
	return l, nil
}

func NewPDelFileOp(strategy string) PDelFileOp {
	strategy = strings.ToLower(strategy)
	switch strategy {
	case PdelStrategyDirect:
		return PDelFileDirectOp{}
	case PdelStrategyTrash:
		return PDelFileTrashOp{}
	default:
		return PDelFileTrashOp{}
	}
}

func FastCheckFileExists(rail miso.Rail, fileId string) error {
	exists, err := fileIdExistCache.Get(rail, fileId, func() (string, error) {
		exists, err := CheckFileExists(fileId)
		if err != nil {
			return "", err
		}
		if exists {
			return "Y", nil
		}
		return "N", nil
	})
	if err != nil {
		return err
	}
	if exists != "Y" {
		return ErrFileNotFound
	}
	return nil
}

// Create random file key for the file
func RandFileKey(rail miso.Rail, name string, fileId string) (string, error) {
	fk := miso.ERand(30)
	err := FastCheckFileExists(rail, fileId)
	if err != nil {
		return "", err
	}

	sby, em := miso.WriteJson(CachedFile{Name: name, FileId: fileId})
	if em != nil {
		return "", fmt.Errorf("failed to marshal to CachedFile, %v", em)
	}
	c := miso.GetRedis().Set("fstore:file:key:"+fk, string(sby), 30*time.Minute)
	return fk, c.Err()
}

// Refresh file key's expiration
func RefreshFileKeyExp(rail miso.Rail, fileKey string) error {
	c := miso.GetRedis().Expire("fstore:file:key:"+fileKey, 30*time.Minute)
	if c.Err() != nil {
		rail.Warnf("Failed to refresh file key expiration, fileKey: %v, %v", fileKey, c.Err())
		return fmt.Errorf("failed to refresh key expiration, %v", c.Err())
	}
	return nil
}

// Resolve CachedFile for the given fileKey
func ResolveFileKey(rail miso.Rail, fileKey string) (bool, CachedFile) {
	var cf CachedFile
	c := miso.GetRedis().Get("fstore:file:key:" + fileKey)
	if c.Err() != nil {
		if errors.Is(c.Err(), redis.Nil) {
			rail.Infof("FileKey not found, %v", fileKey)
		} else {
			rail.Errorf("Failed to find fileKey, %v", c.Err())
		}
		return false, cf
	}

	eu := miso.ParseJson([]byte(c.Val()), &cf)
	if eu != nil {
		rail.Errorf("Failed to unmarshal fileKey, %s, %v", fileKey, c.Err())
		return false, cf
	}
	return true, cf
}

// Adjust ByteRange based on the fileSize
func adjustByteRange(br ByteRange, fileSize int64) (ByteRange, error) {
	if br.End >= fileSize {
		br.End = fileSize - 1
	}

	if br.Start > br.End {
		return br, fmt.Errorf("invalid byte range request, start > end")
	}

	if br.Size() > fileSize {
		return br, fmt.Errorf("invalid byte range request, end - size + 1 > file_size")
	}

	if br.Size() > ByteRangeMaxSize {
		br.End = br.Start + ByteRangeMaxSize - 1
	}

	return br, nil
}

// Stream file by a generated random file key
func StreamFileKey(rail miso.Rail, gc *gin.Context, fileKey string, br ByteRange) error {
	ok, cachedFile := ResolveFileKey(rail, fileKey)
	if !ok {
		return ErrFileNotFound
	}

	ff, err := findDFile(cachedFile.FileId)
	if err != nil {
		return ErrFileNotFound
	}
	if ff.IsDeleted() {
		return ErrFileDeleted
	}

	if e := RefreshFileKeyExp(rail, fileKey); e != nil {
		return e
	}

	var ea error
	br, ea = adjustByteRange(br, ff.Size)
	if ea != nil {
		return ea
	}

	gc.Status(206) // partial content
	gc.Header("Content-Type", "video/mp4")
	gc.Header("Content-Length", strconv.FormatInt(br.Size(), 10))
	gc.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", br.Start, br.End, ff.Size))
	gc.Header("Accept-Ranges", "bytes")

	defer logTransferFilePerf(rail, ff.FileId, br.Size(), time.Now())
	return TransferFile(rail, gc, ff, br)
}

// Download file by a generated random file key
func DownloadFileKey(rail miso.Rail, gc *gin.Context, fileKey string) error {
	ok, cachedFile := ResolveFileKey(rail, fileKey)
	if !ok {
		return ErrFileNotFound
	}

	dname := cachedFile.Name
	inclName := dname == ""

	ff, err := findDFile(cachedFile.FileId)
	if err != nil {
		return ErrFileNotFound
	}
	if ff.IsDeleted() {
		return ErrFileDeleted
	}

	if inclName {
		dname = ff.Name
	}

	gc.Header("Content-Length", strconv.FormatInt(ff.Size, 10))
	gc.Header("Content-Disposition", "attachment; filename=\""+dname+"\"")

	defer logTransferFilePerf(rail, ff.FileId, ff.Size, time.Now())
	return TransferFile(rail, gc, ff, ZeroByteRange())
}

// Download file by file_id
func DownloadFile(rail miso.Rail, gc *gin.Context, fileId string) error {
	if fileId == "" {
		return ErrFileNotFound
	}
	ff, err := findDFile(fileId)
	if err != nil {
		return miso.NewErrf("File not found").WithCode(api.FileNotFound).WithInternalMsg("findDFile failed, fileId: %v, %v", fileId, err)
	}
	if ff.IsDeleted() {
		return ErrFileDeleted
	}
	gc.Header("Content-Length", strconv.FormatInt(ff.Size, 10))
	gc.Header("Content-Disposition", "attachment; filename="+url.QueryEscape(ff.Name))
	defer logTransferFilePerf(rail, ff.FileId, ff.Size, time.Now())
	return TransferFile(rail, gc, ff, ZeroByteRange())
}

func logTransferFilePerf(rail miso.Rail, fileId string, l int64, start time.Time) {
	timeTook := time.Since(start)
	speed := float64(l) / 1e3 / float64(timeTook.Milliseconds())
	rail.Infof("Transferred file '%v', size: '%v', took: '%s', speed: '%.3fmb/s'", fileId, l, timeTook, speed)
}

// Transfer file
func TransferFile(rail miso.Rail, gc *gin.Context, ff DFile, br ByteRange) error {
	p := ff.StoragePath()
	rail.Debugf("Transferring file '%s', path: '%s'", ff.FileId, p)

	// open the file
	f, eo := os.Open(p)
	if eo != nil {
		return fmt.Errorf("failed to open file, %v", eo)
	}
	defer f.Close()

	var et error
	if br.IsZero() {
		// transfer the whole file
		io.Copy(gc.Writer, f)
	} else {
		// jump to start, only transfer a byte range
		if br.Start > 0 {
			f.Seek(br.Start, io.SeekStart)
		}
		_, et = io.CopyN(gc.Writer, f, br.Size())
	}
	return et
}

func NewUploadLock(rail miso.Rail, filename string, size int64, md5 string) *miso.RLock {
	return miso.NewRLockf(rail, "mini-fstore:upload:lock:%v:%v:%v", filename, size, md5)
}

// Upload file and create file record for it
//
// return fileId or any error occured
func UploadFile(rail miso.Rail, rd io.Reader, filename string) (string, error) {
	fileId := GenFileId()
	target := GenStoragePath(fileId)

	rail.Infof("Generated filePath '%s' for fileId '%s'", target, fileId)

	f, ce := os.Create(target)
	if ce != nil {
		return "", fmt.Errorf("failed to create local file, %v", rail)
	}
	defer f.Close()

	size, md5, ecp := CopyChkSum(rd, f)
	if ecp != nil {
		return "", fmt.Errorf("failed to transfer to local file, %v", ecp)
	}

	rlock := NewUploadLock(rail, filename, size, md5)
	if err := rlock.Lock(); err != nil {
		return "", fmt.Errorf("failed to obtain lock, %v", err)
	}
	defer rlock.Unlock()

	duplicateFileId, err := FindDuplicateFile(rail, miso.GetMySQL(), filename, size, md5)
	if err != nil {
		return "", fmt.Errorf("failed to find duplicate file, %v", err)
	}

	// same file is found, save the symbolic link to the previous file instead
	link := ""
	if duplicateFileId != "" {
		os.Remove(target)
		link = duplicateFileId
	}

	ecf := CreateFileRec(rail, CreateFile{
		FileId: fileId,
		Name:   filename,
		Size:   size,
		Md5:    md5,
		Link:   link,
	})
	return fileId, ecf
}

type CreateFile struct {
	FileId string
	Link   string
	Name   string
	Size   int64
	Md5    string
}

// Create file record
func CreateFileRec(rail miso.Rail, c CreateFile) error {
	f := File{
		FileId:  c.FileId,
		Name:    c.Name,
		Status:  StatusNormal,
		Size:    c.Size,
		Md5:     c.Md5,
		Link:    c.Link,
		UplTime: miso.ETime(time.Now()),
	}
	t := miso.GetMySQL().Table("file").Omit("Id", "DelTime").Create(&f)
	if t.Error != nil {
		return t.Error
	}
	return nil
}

func FindDuplicateFile(rail miso.Rail, db *gorm.DB, filename string, size int64, md5 string) (string, error) {
	var fileId string
	t := db.Table("file").
		Select("file_id").
		Where("name = ?", filename).
		Where("size = ?", size).
		Where("md5 = ?", md5).
		Where("status = ?", StatusNormal).
		Limit(1).
		Scan(&fileId)
	if t.Error != nil {
		return "", fmt.Errorf("failed to query duplicate file in db, %v", t.Error)
	}
	return fileId, nil
}

func CheckFileExists(fileId string) (bool, error) {
	var id int
	t := miso.GetMySQL().Raw("select id from file where file_id = ? and status = 'NORMAL'", fileId).Scan(&id)
	if t.Error != nil {
		return false, fmt.Errorf("failed to select file from DB, %w", t.Error)
	}
	return id > 0, nil
}

func CheckAllNormalFiles(fileIds []string) (bool, error) {
	fileIds = miso.Distinct(fileIds)
	var cnt int
	t := miso.GetMySQL().Raw("select count(id) from file where file_id in ? and status = 'NORMAL'", fileIds).Scan(&cnt)
	if t.Error != nil {
		return false, fmt.Errorf("failed to select file from DB, %w", t.Error)
	}
	return cnt == len(fileIds), nil
}

// Find File
func FindFile(db *gorm.DB, fileId string) (File, error) {
	var f File
	t := db.Raw("select * from file where file_id = ?", fileId).Scan(&f)
	if t.Error != nil {
		return f, fmt.Errorf("failed to select file from DB, %w", t.Error)
	}
	return f, nil
}

type DFile struct {
	FileId string
	Link   string
	Size   int64
	Status string
	Name   string
}

// Check if the file is deleted already
func (df *DFile) IsDeleted() bool {
	return df.Status != StatusNormal
}

// Return the actual storage path including symbolic link.
//
// A file could be a symbolic link to another file (using field f.Link).
//
// Be cautious if this path is used to delete/remove files (i.e., it shouldn't).
func (f *DFile) StoragePath() string {
	dfileId := f.FileId
	if f.Link != "" {
		dfileId = f.Link
	}
	return GenStoragePath(dfileId)
}

func findDFile(fileId string) (DFile, error) {
	var df DFile
	t := miso.GetMySQL().
		Select("file_id, size, status, name, link").
		Table("file").
		Where("file_id = ?", fileId).
		Scan(&df)

	if err := t.Error; err != nil {
		return df, err
	}
	if t.RowsAffected < 1 {
		return df, ErrFileNotFound
	}
	return df, nil
}

// Delete file logically by changing it's status
func LDelFile(rail miso.Rail, db *gorm.DB, fileId string) error {
	fileId = strings.TrimSpace(fileId)
	if fileId == "" {
		return miso.NewErrf("fileId is required").WithCode(api.InvalidRequest)
	}

	lock := miso.NewRLock(rail, FileLockKey(fileId))
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()

	f, er := FindFile(db, fileId)
	if er != nil {
		return miso.NewErrf("Unknown error").WithCode(api.UnknownError).
			WithInternalMsg("FindFile failed, %v", er)
	}

	if f.IsZero() {
		return ErrFileNotFound
	}

	if f.IsDeleted() {
		return ErrFileDeleted
	}

	t := db.Exec("update file set status = ?, log_del_time = ? where file_id = ?", StatusLogicDel, time.Now(), fileId)
	if t.Error != nil {
		return miso.NewErrf("Unknown error").WithCode(api.UnknownError).WithInternalMsg("Failed to update file, %v", t.Error)
	}
	return nil
}

// List logically deleted files
func ListLDelFile(rail miso.Rail, idOffset int64, limit int) ([]File, error) {
	var l []File = []File{}

	t := miso.GetMySQL().
		Raw("select * from file where id > ? and status = ? limit ?", idOffset, StatusLogicDel, limit).
		Scan(&l)
	if t.Error != nil {
		return nil, fmt.Errorf("failed to list logically deleted files, %v", t.Error)
	}

	return l, nil
}

// Mark file as physically deleted by changing it's status
func PhyDelFile(rail miso.Rail, db *gorm.DB, fileId string, op PDelFileOp) error {
	fileId = strings.TrimSpace(fileId)
	if fileId == "" {
		return miso.NewErrf("fileId is required").WithCode(api.InvalidRequest)
	}

	_, e := miso.RLockRun(rail, FileLockKey(fileId), func() (any, error) {

		f, er := FindFile(db, fileId)
		if er != nil {
			return nil, miso.NewErrf("Unknown error").WithCode(api.UnknownError).WithInternalMsg("FindFile failed, %v", er)
		}

		if f.IsZero() {
			return nil, ErrFileDeleted
		}

		if !f.IsLogiDeleted() {
			return nil, nil
		}

		// the file may be pointed by another symbolic file
		// before we delete it, we need to make sure that it's not pointed
		// by other files
		var refId int
		if err := miso.GetMySQL().
			Raw("select id from file where link = ? and status = ? limit 1", f.FileId, StatusNormal).
			Scan(&refId).Error; err != nil {
			return nil, fmt.Errorf("failed to check symbolic link, fileId: %v, %v", f.FileId, err)
		}
		if refId > 0 { // link exists, we cannot really delete it
			rail.Infof("File %v is still symbolically linked by other files, cannot be removed yet", fileId)
			return nil, nil
		}

		if ed := op.delete(rail, fileId); ed != nil {
			return nil, ed
		}

		t := miso.GetMySQL().
			Exec("update file set status = ?, phy_del_time = ? where file_id = ?", StatusPhysicDel, time.Now(), fileId)
		if t.Error != nil {
			return nil, miso.NewErrf("Unknown error").WithCode(api.UnknownError).WithInternalMsg("Failed to update file, %v", t.Error)
		}

		return nil, nil
	})
	return e
}

// Concatenate file's redis lock key
func FileLockKey(fileId string) string {
	return "fstore:file:" + fileId
}

func SanitizeStorage(rail miso.Rail) error {
	dirPath := miso.GetPropStr(PropStorageDir)
	files, e := os.ReadDir(dirPath)
	if e != nil {
		if os.IsNotExist(e) {
			return nil
		}
		return fmt.Errorf("failed to read dir, %v", e)
	}
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	rail.Infof("Found %v files", len(files))
	threshold := time.Now().Add(-6 * time.Hour)
	for _, f := range files {
		fi, e := f.Info()
		if e != nil {
			return fmt.Errorf("failed to read file info, %v", e)
		}
		fileId := fi.Name()

		// make sure the file is not being uploaded recently, and we don't accidentally 'moved' a new file
		if fi.ModTime().After(threshold) {
			continue
		}

		// check if the file is in database
		f, e := FindFile(miso.GetMySQL(), fileId)
		if e != nil {
			return fmt.Errorf("failed to find file from db, %v", e)
		}

		if !f.IsZero() {
			continue // valid file
		}

		// file record is not found, file should be moved to trash dir
		frm := dirPath + fileId
		to := GenTrashPath(fileId)

		if miso.GetPropBool(PropSanitizeStorageTaskDryRun) { // dry-run
			rail.Infof("Sanitizing storage, (dry-run) will rename file from %s to %s", frm, to)
		} else {
			if e := os.Rename(frm, to); e != nil {
				if os.IsNotExist(e) {
					rail.Infof("File has been deleted, file: %s", frm)
					continue
				}
				rail.Errorf("Sanitizing storage, failed to rename file from %s to %s, %v", frm, to, e)
			}
			rail.Infof("Sanitizing storage, renamed file from %s to %s", frm, to)
		}
	}
	return nil
}

// Trigger unzip file pipeline.
//
// Unzipping is asynchrounous, the unzipped files are saved in mini-fstore, and the final result is replied to the specified event bus.
func TriggerUnzipFilePipeline(rail miso.Rail, db *gorm.DB, req api.UnzipFileReq) error {
	f, e := FindFile(db, req.FileId)
	if e != nil {
		return ErrFileNotFound
	}
	if f.IsDeleted() {
		return ErrFileDeleted
	}

	lname := strings.ToLower(f.Name)
	if !strings.HasSuffix(lname, ".zip") {
		return miso.NewErrf("Not a zip file").WithCode(api.IllegalFormat)
	}

	err := miso.PubEventBus(rail, UnzipFileEvent(req), UnzipPipelineEventBus)
	if err != nil {
		return fmt.Errorf("failed to send event, req: %+v, %v", req, err)
	}
	return nil
}

func UnzipFile(rail miso.Rail, db *gorm.DB, evt UnzipFileEvent) ([]SavedZipEntry, error) {
	defer miso.TimeOp(rail, time.Now(), fmt.Sprintf("Unzip file %v", evt.FileId))

	rail.Infof("About to unpack zip file, fileId: %v", evt.FileId)
	f, e := FindFile(db, evt.FileId)
	if e != nil {
		rail.Infof("file is not found, %v", evt.FileId)
		return nil, nil
	}
	if f.IsDeleted() {
		rail.Infof("file is deleted, %v", evt.FileId)
		return nil, nil
	}

	lname := strings.ToLower(f.Name)
	if !strings.HasSuffix(lname, ".zip") {
		rail.Infof("file is not a zip file, %v", evt.FileId)
		return nil, nil
	}

	tempDir := miso.GetPropStr(PropTempDir) + "/" + evt.FileId + "_" + miso.RandNum(5)
	if err := os.MkdirAll(tempDir, miso.DefFileMode); err != nil {
		return nil, fmt.Errorf("failed to MkdirAll for tempDir %v, %w", tempDir, err)
	}

	defer os.RemoveAll(tempDir)
	rail.Infof("Made temp dir: %v", tempDir)

	entries, err := UnpackZip(rail, f, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to unzip file, fileId: %v, filename: %v, %v", f.FileId, f.Name, err)
	}
	rail.Infof("Unpacked file %v (%v), entries: %+v", f.FileId, f.Name, entries)

	saved, err := SaveZipFiles(rail, db, entries)
	rail.Infof("Saved zip entries %v (%v), saved: %+v, err: %v", f.FileId, f.Name, saved, err)
	return saved, err
}

type UnpackedZipEntry struct {
	Md5  string
	Name string
	Path string
	Size int64
}

func UnpackZip(rail miso.Rail, f File, tempDir string) ([]UnpackedZipEntry, error) {
	zipPath := f.StoragePath()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file, %w", err)
	}
	defer r.Close()

	// guessing that most of the time we have at least 15 entries in a zip
	entries := make([]UnpackedZipEntry, 0, 15)

	for _, f := range r.File {
		entryReader, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip entry file %v, %w", f.Name, err)
		}

		tempPath := tempDir + "/" + miso.GenIdP("ZIPENTRY")
		tempFile, err := miso.ReadWriteFile(tempPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file for zip entry file, %v, %w", f.Name, err)
		}

		// copy from zip entry to temp file
		size, md5, err := CopyChkSum(entryReader, tempFile)

		entryReader.Close()
		tempFile.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to copy entry file to temp file, %v, %w", f.Name, err)
		}

		entries = append(entries, UnpackedZipEntry{
			Name: f.Name,
			Md5:  md5,
			Size: size,
			Path: tempPath,
		})
	}
	return entries, nil
}

func SaveZipFiles(rail miso.Rail, db *gorm.DB, entries []UnpackedZipEntry) ([]SavedZipEntry, error) {
	saved := make([]SavedZipEntry, 0, len(entries))
	for _, et := range entries {
		ent, err := SaveZipFile(rail, db, et)
		if err != nil {
			return nil, fmt.Errorf("failed to save zip entry file, entry: %+v, %v", et, err)
		}
		rail.Infof("Saved entry file %v to %v", et.Name, ent)
		saved = append(saved, ent)
	}
	return saved, nil
}

type SavedZipEntry struct {
	Md5    string
	Name   string
	Size   int64
	FileId string
}

func SaveZipFile(rail miso.Rail, db *gorm.DB, entry UnpackedZipEntry) (SavedZipEntry, error) {
	rlock := NewUploadLock(rail, entry.Name, entry.Size, entry.Md5)
	if err := rlock.Lock(); err != nil {
		return SavedZipEntry{}, fmt.Errorf("failed to obtain lock, %v", err)
	}
	defer rlock.Unlock()

	duplicateFileId, err := FindDuplicateFile(rail, db, entry.Name, entry.Size, entry.Md5)
	if err != nil {
		return SavedZipEntry{}, fmt.Errorf("failed to find duplicate file, %v", err)
	}

	fileId := GenFileId()
	link := ""
	if duplicateFileId != "" {
		// same file is found, save the symbolic link to the previous file instead
		// the temporary entry file will be removed anyway
		link = duplicateFileId
		rail.Infof("Found duplicate upload, create symbolic link for %v to %v", entry.Name, duplicateFileId)
	} else {
		// file is not found, move the zip entry file to the storage directory
		storagePath := GenStoragePath(fileId)
		err := os.Rename(entry.Path, storagePath)
		if err != nil {
			return SavedZipEntry{}, fmt.Errorf("failed to move zip entry file from %v to %v, %v", entry.Path, storagePath, err)
		}
	}

	err = CreateFileRec(rail, CreateFile{
		FileId: fileId,
		Name:   entry.Name,
		Size:   entry.Size,
		Md5:    entry.Md5,
		Link:   link,
	})
	if err != nil {
		return SavedZipEntry{}, fmt.Errorf("failled to create file record for zip entry, %v", err)
	}

	return SavedZipEntry{
		Md5:    entry.Md5,
		Name:   entry.Name,
		Size:   entry.Size,
		FileId: fileId,
	}, err
}
