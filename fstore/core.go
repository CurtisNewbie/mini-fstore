package fstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

const (
	FILE_ID_PREFIX = "file_" // prefix of file_id

	STATUS_NORMAL     = "NORMAL"  // file.status - normal
	STATUS_LOGIC_DEL  = "LOG_DEL" // file.status - logically deletedy
	STATUS_PHYSIC_DEL = "PHY_DEL" // file.status - physically deletedy

	PDEL_STRAT_DIRECT = "direct" // file delete strategy - direct
	PDEL_STRAT_TRASH  = "trash"  // file delete strategy - trash

	BYTE_RANGE_MAX_SIZE = 30_000_000 // 30 mb
)

var (
	_trashMkdir   int32 = 0 // did we do mkdir for trash dir; atomic value, 0 - false, 1 - true
	_storageMkdir int32 = 0 // did we do mkdir for storage dir; atomic value, 0 - false, 1 - true

	ErrFileNotFound = miso.NewErrCode(FILE_NOT_FOUND, "File is not found")
	ErrFileDeleted  = miso.NewErrCode(FILE_DELETED, "File has been deleted already")

	fileIdExistCache = miso.NewLazyRCache("fstore:fileid:exist:", 10*time.Minute, func(rail miso.Rail, key string) (string, error) {
		exists, err := CheckFileExists(key)
		if err != nil {
			return "", err
		}
		if exists {
			return "Y", nil
		}
		return "N", nil
	})
)

func init() {
	miso.SetDefProp(PROP_STORAGE_DIR, "./storage")
	miso.SetDefProp(PROP_TRASH_DIR, "./trash")
	miso.SetDefProp(PROP_PDEL_STRATEGY, PDEL_STRAT_TRASH)
	miso.SetDefProp(PROP_SANITIZE_STORAGE_TASK_DRY_RUN, false)
}

type DownloadFileReq struct {
	FileId   string `form:"fileId"`
	Filename string `form:"filename"`
}

type DeleteFileReq struct {
	FileId string `form:"fileId" valid:"notEmpty"`
}

type BatchGenFileKeyReq struct {
	Items []GenFileKeyReq `json:"items"`
}

type GenFileKeyReq struct {
	FileId   string `json:"fileId"`
	Filename string `json:"filename"`
}

type GenFileKeyResp struct {
	FileId  string `json:"fileId"`
	TempKey string `json:"tempKey"`
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

		Implmentation should detect whether the file still exists before undertaking deletion. If file has been deleted, nil error should be returned
	*/
	delete(r miso.Rail, fileId string) error
}

// The 'direct' implementation of of PDelFileOp, files are deleted directly
type PDelFileDirectOp struct {
}

func (p PDelFileDirectOp) delete(rail miso.Rail, fileId string) error {
	file, e := GenStoragePath(rail, fileId)
	if e != nil {
		return fmt.Errorf("failed to GenFilePath, %v", e)
	}
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
	frm, e := GenStoragePath(rail, fileId)
	if e != nil {
		return fmt.Errorf("failed to GenFilePath, %v", e)
	}

	to, e := GenTrashPath(rail, fileId)
	if e != nil {
		return fmt.Errorf("failed to GenTrashPath, %v", e)
	}

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

type CreateFile struct {
	FileId string
	Name   string
	Size   int64
	Md5    string
}

type File struct {
	Id         int64       `json:"id"`
	FileId     string      `json:"fileId"`
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
	return f.Status != STATUS_NORMAL
}

// Check if the file is logically already
func (f *File) IsLogiDeleted() bool {
	return f.Status == STATUS_LOGIC_DEL
}

// Generate random file_id
func GenFileId() string {
	return miso.GenIdP(FILE_ID_PREFIX)
}

// Generate file path
//
// Property `fstore.storage.dir` is used
func GenStoragePath(rail miso.Rail, fileId string) (string, error) {
	dir := miso.GetPropStr(PROP_STORAGE_DIR)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	doMkdir := atomic.CompareAndSwapInt32(&_storageMkdir, 0, 1)
	if doMkdir {
		em := os.MkdirAll(dir, os.ModePerm)
		if em != nil {
			rail.Errorf("os.MkdirAll failed while trying to GenFilePath, %v", em)
			return "", fmt.Errorf("failed to MkdirAll, %v", em)
		}
	}

	return dir + fileId, nil
}

// Generate file path for trashed file
//
// Property `fstore.trash.dir` is used
func GenTrashPath(rail miso.Rail, fileId string) (string, error) {
	dir := miso.GetPropStr(PROP_TRASH_DIR)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	doMkdir := atomic.CompareAndSwapInt32(&_trashMkdir, 0, 1)
	if doMkdir {
		em := os.MkdirAll(dir, os.ModePerm)
		if em != nil {
			rail.Errorf("os.MkdirAll failed while trying to GenTrashPath, %v", em)
			return "", fmt.Errorf("failed to MkdirAll, %v", em)
		}
	}
	return dir + fileId, nil
}

/*
List logically deleted files, and based on the configured strategy, deleted them 'physically'.

This func reads property 'fstore.pdelete.strategy'.

If strategy is 'direct', files are deleted directly.

If strategy is 'trash' (default), files are moved to 'trash' directory, which is specified in property 'fstore.trash.dir'
*/
func BatchPhyDelFiles(rail miso.Rail) error {
	start := time.Now()
	defer miso.TimeOp(rail, start, "BatchPhyDelFiles")

	before := start.Add(-1 * time.Hour) // only delete files that are logically deleted 1 hour ago

	var l []string
	var le error
	l, le = listPendingPhyDelFiles(rail, before)
	if le != nil {
		return le
	}
	if l == nil {
		return nil
	}

	strat := miso.GetPropStr(PROP_PDEL_STRATEGY)
	delFileOp := NewPDelFileOp(strat)

	for _, fileId := range l {
		if e := PhyDelFile(rail, fileId, delFileOp); e != nil {
			rail.Errorf("Failed to PhyDelFile, strategy: %v, fileId: %s, %v", strat, fileId, e)
		}
	}
	return nil
}

func listPendingPhyDelFiles(rail miso.Rail, beforeLogDelTime time.Time) ([]string, error) {
	defer miso.TimeOp(rail, time.Now(), "listPendingPhyDelFiles")

	var l []string
	tx := miso.GetMySQL().
		Raw("select file_id from file where status = ? and log_del_time <= ? limit 5000", STATUS_LOGIC_DEL, beforeLogDelTime).
		Scan(&l)

	if e := tx.Error; e != nil {
		rail.Errorf("Failed to list LDel files, %v", e)
		return nil, e
	}
	if l == nil {
		rail.Info("No files to delete")
	}

	return l, nil
}

func NewPDelFileOp(strategy string) PDelFileOp {
	strategy = strings.ToLower(strategy)
	switch strategy {
	case PDEL_STRAT_DIRECT:
		return PDelFileDirectOp{}
	case PDEL_STRAT_TRASH:
		return PDelFileTrashOp{}
	default:
		return PDelFileTrashOp{}
	}
}

// Create random file key for the file
func RandFileKey(rail miso.Rail, name string, fileId string) (string, error) {
	s := miso.ERand(30)
	exists, err := fileIdExistCache.Get(rail, fileId)
	if err != nil {
		return "", err
	}
	if exists != "Y" {
		return "", ErrFileNotFound
	}

	cf := CachedFile{Name: name, FileId: fileId}
	sby, em := json.Marshal(cf)
	if em != nil {
		return "", fmt.Errorf("failed to marshal to CachedFile, %v", em)
	}
	c := miso.GetRedis().Set("fstore:file:key:"+s, string(sby), 30*time.Minute)
	return s, c.Err()
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

	eu := json.Unmarshal([]byte(c.Val()), &cf)
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

	if br.Size() > BYTE_RANGE_MAX_SIZE {
		br.End = br.Start + BYTE_RANGE_MAX_SIZE - 1
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
	return TransferFile(rail, gc, ff.FileId, br)
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
	return TransferFile(rail, gc, ff.FileId, ZeroByteRange())
}

// Download file by file_id
func DownloadFile(rail miso.Rail, gc *gin.Context, fileId string) error {
	if fileId == "" {
		return ErrFileNotFound
	}
	ff, err := findDFile(fileId)
	if err != nil {
		return miso.NewErrCode(FILE_NOT_FOUND, "Unable to find file", "findDFile failed, fileId: %v, %v", fileId, err)
	}
	if ff.IsDeleted() {
		return ErrFileDeleted
	}
	gc.Header("Content-Length", strconv.FormatInt(ff.Size, 10))
	gc.Header("Content-Disposition", "attachment; filename="+url.QueryEscape(ff.Name))
	defer logTransferFilePerf(rail, ff.FileId, ff.Size, time.Now())
	return TransferFile(rail, gc, ff.FileId, ZeroByteRange())
}

func logTransferFilePerf(rail miso.Rail, fileId string, l int64, start time.Time) {
	timeTook := time.Since(start)
	speed := float64(l) / 1e3 / float64(timeTook.Milliseconds())
	rail.Infof("Transferred file '%v', size: '%v', took: '%s', speed: '%.3fmb/s'", fileId, l, timeTook, speed)
}

// Transfer file
func TransferFile(rail miso.Rail, gc *gin.Context, fileId string, br ByteRange) error {

	p, eg := GenStoragePath(rail, fileId)
	if eg != nil {
		return fmt.Errorf("failed to generate file path, %v", eg)
	}
	rail.Debugf("Transferring file '%s', path: '%s'", fileId, p)

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

// Upload file and create file record for it
//
// return fileId or any error occured
func UploadFile(rail miso.Rail, rd io.Reader, filename string) (string, error) {
	fileId := GenFileId()
	target, eg := GenStoragePath(rail, fileId)
	if eg != nil {
		return "", fmt.Errorf("failed to generate file path, %v", eg)
	}

	rail.Infof("Generated filePath '%s' for fileId '%s'", target, fileId)

	f, ce := os.Create(target)
	if ce != nil {
		return "", fmt.Errorf("failed to create local file, %v", rail)
	}

	size, md5, ecp := CopyChkSum(rd, f)
	if ecp != nil {
		return "", fmt.Errorf("failed to transfer to local file, %v", ecp)
	}

	// if size < 1 {
	// 	return "", miso.NewErr("You cannot upload empty file")
	// }

	ecf := CreateFileRec(rail, CreateFile{
		FileId: fileId,
		Name:   filename,
		Size:   size,
		Md5:    md5,
	})
	return fileId, ecf
}

// Create file record
func CreateFileRec(rail miso.Rail, c CreateFile) error {
	f := File{
		FileId:  c.FileId,
		Name:    c.Name,
		Status:  STATUS_NORMAL,
		Size:    c.Size,
		Md5:     c.Md5,
		UplTime: miso.ETime(time.Now()),
	}
	t := miso.GetMySQL().Table("file").Omit("Id", "DelTime").Create(&f)
	if t.Error != nil {
		return t.Error
	}
	return nil
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
func FindFile(fileId string) (File, error) {
	var f File
	t := miso.GetMySQL().Raw("select * from file where file_id = ?", fileId).Scan(&f)
	if t.Error != nil {
		return f, fmt.Errorf("failed to select file from DB, %w", t.Error)
	}
	return f, nil
}

type DFile struct {
	FileId string
	Size   int64
	Status string
	Name   string
}

// Check if the file is deleted already
func (df DFile) IsDeleted() bool {
	return df.Status != STATUS_NORMAL
}

func findDFile(fileId string) (DFile, error) {
	var df DFile
	t := miso.GetMySQL().
		Select("file_id, size, status, name").
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
func LDelFile(rail miso.Rail, fileId string) error {
	fileId = strings.TrimSpace(fileId)
	if fileId == "" {
		return miso.NewErrCode(INVALID_REQUEST, "fileId is required")
	}

	_, e := miso.RLockRun(rail, FileLockKey(fileId), func() (any, error) {
		f, er := FindFile(fileId)
		if er != nil {
			return nil, miso.NewErrCode(UNKNOWN_ERROR, er.Error())
		}

		if f.IsZero() {
			return nil, ErrFileNotFound
		}

		if f.IsDeleted() {
			return nil, ErrFileDeleted
		}

		t := miso.GetMySQL().Exec("update file set status = ?, log_del_time = ? where file_id = ?", STATUS_LOGIC_DEL, time.Now(), fileId)
		if t.Error != nil {
			return nil, miso.NewErrCode(UNKNOWN_ERROR, fmt.Sprintf("Failed to update file, %v", t.Error))
		}

		return nil, nil
	})
	return e
}

// List logically deleted files
func ListLDelFile(rail miso.Rail, idOffset int64, limit int) ([]File, error) {
	var l []File = []File{}

	t := miso.GetMySQL().
		Raw("select * from file where id > ? and status = ? limit ?", idOffset, STATUS_LOGIC_DEL, limit).
		Scan(&l)
	if t.Error != nil {
		return nil, fmt.Errorf("failed to list logically deleted files, %v", t.Error)
	}

	return l, nil
}

// Mark file as physically deleted by changing it's status
func PhyDelFile(rail miso.Rail, fileId string, op PDelFileOp) error {
	fileId = strings.TrimSpace(fileId)
	if fileId == "" {
		return miso.NewErrCode(INVALID_REQUEST, "fileId is required")
	}

	_, e := miso.RLockRun(rail, FileLockKey(fileId), func() (any, error) {
		f, er := FindFile(fileId)
		if er != nil {
			return nil, miso.NewErrCode(UNKNOWN_ERROR, er.Error())
		}

		if f.IsZero() {
			return nil, ErrFileDeleted
		}

		if !f.IsLogiDeleted() {
			return nil, nil
		}

		if ed := op.delete(rail, fileId); ed != nil {
			return nil, ed
		}

		t := miso.GetMySQL().
			Exec("update file set status = ?, phy_del_time = ? where file_id = ?", STATUS_PHYSIC_DEL, time.Now(), fileId)
		if t.Error != nil {
			return nil, miso.NewErrCode(UNKNOWN_ERROR, fmt.Sprintf("Failed to update file, %v", t.Error))
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
	dirPath := miso.GetPropStr(PROP_STORAGE_DIR)
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
		f, e := FindFile(fileId)
		if e != nil {
			return fmt.Errorf("failed to find file from db, %v", e)
		}

		if !f.IsZero() {
			continue // valid file
		}

		// file record is not found, file should be moved to trash dir
		frm := dirPath + fileId
		to, e := GenTrashPath(rail, fileId)
		if e != nil {
			return fmt.Errorf("failed to GenTrashPath, %v", e)
		}

		if miso.GetPropBool(PROP_SANITIZE_STORAGE_TASK_DRY_RUN) { // dry-run
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
