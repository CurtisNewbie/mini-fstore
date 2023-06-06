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

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/mysql"
	red "github.com/curtisnewbie/gocommon/redis"
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

	ErrFileNotFound = common.NewWebErrCode(FILE_NOT_FOUND, "File is not found")
	ErrFileDeleted  = common.NewWebErrCode(FILE_DELETED, "File has been deleted already")
)

func init() {
	common.SetDefProp(PROP_STORAGE_DIR, "./storage")
	common.SetDefProp(PROP_TRASH_DIR, "./trash")
	common.SetDefProp(PROP_PDEL_STRATEGY, PDEL_STRAT_TRASH)
	common.SetDefProp(PROP_SANITIZE_STORAGE_TASK_DRY_RUN, false)
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
	delete(c common.ExecContext, fileId string) error
}

// The 'direct' implementation of of PDelFileOp, files are deleted directly
type PDelFileDirectOp struct {
}

func (p PDelFileDirectOp) delete(c common.ExecContext, fileId string) error {
	file, e := GenStoragePath(c, fileId)
	if e != nil {
		return fmt.Errorf("failed to GenFilePath, %v", e)
	}
	er := os.Remove(file)
	if er != nil {
		if os.IsNotExist(er) {
			c.Log.Infof("File has been deleted, file: %s", file)
			return nil
		}

		c.Log.Errorf("Failed to delete file, file: %s, %v", file, er)
		return er
	}

	return nil
}

// The 'trash' implementation of of PDelFileOp, files are deleted directly
type PDelFileTrashOp struct {
}

func (p PDelFileTrashOp) delete(c common.ExecContext, fileId string) error {
	frm, e := GenStoragePath(c, fileId)
	if e != nil {
		return fmt.Errorf("failed to GenFilePath, %v", e)
	}

	to, e := GenTrashPath(c, fileId)
	if e != nil {
		return fmt.Errorf("failed to GenTrashPath, %v", e)
	}

	if e := os.Rename(frm, to); e != nil {
		if os.IsNotExist(e) {
			c.Log.Infof("File has been deleted, file: %s", frm)
			return nil
		}
		return fmt.Errorf("failed to rename file from %s, to %s, %v", frm, to, e)
	}

	c.Log.Infof("Renamed file from %s, to %s", frm, to)
	return nil
}

type CreateFile struct {
	FileId string
	Name   string
	Size   int64
	Md5    string
}

type File struct {
	Id         int64         `json:"id"`
	FileId     string        `json:"fileId"`
	Name       string        `json:"name"`
	Status     string        `json:"status"`
	Size       int64         `json:"size"`
	Md5        string        `json:"md5"`
	UplTime    common.ETime  `json:"uplTime"`
	LogDelTime *common.ETime `json:"logDelTime"`
	PhyDelTime *common.ETime `json:"phyDelTime"`
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
	return common.GenIdP(FILE_ID_PREFIX)
}

// Generate file path
//
// Property `fstore.storage.dir` is used
func GenStoragePath(ec common.ExecContext, fileId string) (string, error) {
	dir := common.GetPropStr(PROP_STORAGE_DIR)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	doMkdir := atomic.CompareAndSwapInt32(&_storageMkdir, 0, 1)
	if doMkdir {
		em := os.MkdirAll(dir, os.ModePerm)
		if em != nil {
			ec.Log.Errorf("os.MkdirAll failed while trying to GenFilePath, %v", em)
			return "", fmt.Errorf("failed to MkdirAll, %v", em)
		}
	}

	return dir + fileId, nil
}

// Generate file path for trashed file
//
// Property `fstore.trash.dir` is used
func GenTrashPath(ec common.ExecContext, fileId string) (string, error) {
	dir := common.GetPropStr(PROP_TRASH_DIR)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	doMkdir := atomic.CompareAndSwapInt32(&_trashMkdir, 0, 1)
	if doMkdir {
		em := os.MkdirAll(dir, os.ModePerm)
		if em != nil {
			ec.Log.Errorf("os.MkdirAll failed while trying to GenTrashPath, %v", em)
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
func BatchPhyDelFiles(c common.ExecContext) error {
	start := time.Now()
	defer common.TimeOp(c, start, "BatchPhyDelFiles")

	before := start.Add(-1 * time.Hour) // only delete files that are logically deleted 1 hour ago

	var l []string
	var le error
	l, le = listPendingPhyDelFiles(c, before)
	if le != nil {
		return le
	}
	if l == nil {
		return nil
	}

	strat := common.GetPropStr(PROP_PDEL_STRATEGY)
	delFileOp := NewPDelFileOp(strat)

	for _, fileId := range l {
		if e := PhyDelFile(c, fileId, delFileOp); e != nil {
			c.Log.Errorf("Failed to PhyDelFile, strategy: %v, fileId: %s, %v", strat, fileId, e)
		}
	}
	return nil
}

func listPendingPhyDelFiles(c common.ExecContext, beforeLogDelTime time.Time) ([]string, error) {
	defer common.TimeOp(c, time.Now(), "listPendingPhyDelFiles")

	var l []string
	tx := mysql.GetMySql().
		Raw("select file_id from file where status = ? and log_del_time <= ? limit 5000", STATUS_LOGIC_DEL, beforeLogDelTime).
		Scan(&l)

	if e := tx.Error; e != nil {
		c.Log.Errorf("Failed to list LDel files, %v", e)
		return nil, e
	}
	if l == nil {
		c.Log.Info("No files to delete")
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
func RandFileKey(ec common.ExecContext, name string, fileId string) (string, error) {
	s, er := common.ERand(30)
	if er != nil {
		return "", er
	}
	ff, err := FindFile(fileId)
	if err != nil {
		return "", err
	}
	if ff.IsZero() {
		return "", ErrFileNotFound
	}

	if ff.IsDeleted() {
		return "", ErrFileDeleted
	}

	cf := CachedFile{Name: name, FileId: fileId}
	sby, em := json.Marshal(cf)
	if em != nil {
		return "", fmt.Errorf("failed to marshal to CachedFile, %v", em)
	}
	c := red.GetRedis().Set("fstore:file:key:"+s, string(sby), 30*time.Minute)
	return s, c.Err()
}

// Refresh file key's expiration
func RefreshFileKeyExp(ec common.ExecContext, fileKey string) error {
	c := red.GetRedis().Expire("fstore:file:key:"+fileKey, 30*time.Minute)
	if c.Err() != nil {
		ec.Log.Warnf("Failed to refresh file key expiration, fileKey: %v, %v", fileKey, c.Err())
		return fmt.Errorf("failed to refresh key expiration, %v", c.Err())
	}
	return nil
}

// Resolve CachedFile for the given fileKey
func ResolveFileKey(ec common.ExecContext, fileKey string) (bool, CachedFile) {
	var cf CachedFile
	c := red.GetRedis().Get("fstore:file:key:" + fileKey)
	if c.Err() != nil {
		if errors.Is(c.Err(), redis.Nil) {
			ec.Log.Infof("FileKey not found, %v", fileKey)
		} else {
			ec.Log.Errorf("Failed to find fileKey, %v", c.Err())
		}
		return false, cf
	}

	eu := json.Unmarshal([]byte(c.Val()), &cf)
	if eu != nil {
		ec.Log.Errorf("Failed to unmarshal fileKey, %s, %v", fileKey, c.Err())
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
func StreamFileKey(ec common.ExecContext, gc *gin.Context, fileKey string, br ByteRange) error {
	ok, cachedFile := ResolveFileKey(ec, fileKey)
	if !ok {
		return ErrFileNotFound
	}

	ff, err := FindFile(cachedFile.FileId)
	if err != nil {
		return ErrFileNotFound
	}

	if e := RefreshFileKeyExp(ec, fileKey); e != nil {
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
	return TransferFile(ec, gc.Writer, ff, br)
}

// Download file by a generated random file key
func DownloadFileKey(ec common.ExecContext, gc *gin.Context, fileKey string) error {
	ok, cachedFile := ResolveFileKey(ec, fileKey)
	if !ok {
		return ErrFileNotFound
	}

	ff, err := FindFile(cachedFile.FileId)
	if err != nil {
		return ErrFileNotFound
	}

	dname := cachedFile.Name
	if dname == "" {
		dname = ff.Name
	}

	gc.Header("Content-Length", strconv.FormatInt(ff.Size, 10))
	gc.Header("Content-Disposition", "attachment; filename="+url.QueryEscape(dname))
	return TransferFile(ec, gc.Writer, ff, ZeroByteRange())
}

// Transfer file
func TransferFile(ec common.ExecContext, w io.Writer, ff File, br ByteRange) error {
	start := time.Now()
	fileId := ff.FileId

	if ff.IsDeleted() {
		return ErrFileDeleted
	}

	p, eg := GenStoragePath(ec, fileId)
	if eg != nil {
		return fmt.Errorf("failed to generate file path, %v", eg)
	}
	ec.Log.Infof("Transferring file '%s', path: '%s'", fileId, p)

	f, eo := os.Open(p)
	if eo != nil {
		return fmt.Errorf("failed to open file, %v", eo)
	}

	var l int64
	var et error
	if br.IsZero() {
		l, et = io.Copy(w, f)
	} else {
		// jump to start
		f.Seek(br.Start, io.SeekStart)
		l, et = io.CopyN(w, f, br.Size())
	}

	if et != nil {
		return fmt.Errorf("failed to transfer file, %v", et)
	}
	ec.Log.Infof("Transferred file '%v', size: '%v', took: '%s'", fileId, l, time.Since(start))
	return nil
}

// Upload file and create file record for it
//
// return fileId or any error occured
func UploadFile(ec common.ExecContext, rd io.Reader, filename string) (string, error) {
	fileId := GenFileId()
	target, eg := GenStoragePath(ec, fileId)
	if eg != nil {
		return "", fmt.Errorf("failed to generate file path, %v", eg)
	}

	ec.Log.Infof("Generated filePath '%s' for fileId '%s'", target, fileId)

	f, ce := os.Create(target)
	if ce != nil {
		return "", fmt.Errorf("failed to create local file, %v", ec)
	}

	size, md5, ecp := CopyChkSum(rd, f)
	if ecp != nil {
		return "", fmt.Errorf("failed to transfer to local file, %v", ecp)
	}

	ecf := CreateFileRec(ec, CreateFile{
		FileId: fileId,
		Name:   filename,
		Size:   size,
		Md5:    md5,
	})
	return fileId, ecf
}

// Create file record
func CreateFileRec(ec common.ExecContext, c CreateFile) error {
	f := File{
		FileId:  c.FileId,
		Name:    c.Name,
		Status:  STATUS_NORMAL,
		Size:    c.Size,
		Md5:     c.Md5,
		UplTime: common.ETime(time.Now()),
	}
	t := mysql.GetMySql().Table("file").Omit("Id", "DelTime").Create(&f)
	if t.Error != nil {
		return t.Error
	}
	return nil
}

// Find File
func FindFile(fileId string) (File, error) {
	var f File
	t := mysql.GetMySql().Raw("select * from file where file_id = ?", fileId).Scan(&f)
	if t.Error != nil {
		return f, fmt.Errorf("failed to select file from DB, %w", t.Error)
	}
	return f, nil
}

// Delete file logically by changing it's status
func LDelFile(ec common.ExecContext, fileId string) error {
	fileId = strings.TrimSpace(fileId)
	if fileId == "" {
		return common.NewWebErrCode(INVALID_REQUEST, "fileId is required")
	}

	_, e := red.RLockRun(ec, FileLockKey(fileId), func() (any, error) {
		f, er := FindFile(fileId)
		if er != nil {
			return nil, common.NewWebErrCode(UNKNOWN_ERROR, er.Error())
		}

		if f.IsZero() {
			return nil, ErrFileNotFound
		}

		if f.IsDeleted() {
			return nil, ErrFileDeleted
		}

		t := mysql.GetMySql().Exec("update file set status = ?, log_del_time = ? where file_id = ?", STATUS_LOGIC_DEL, time.Now(), fileId)
		if t.Error != nil {
			return nil, common.NewWebErrCode(UNKNOWN_ERROR, fmt.Sprintf("Failed to update file, %v", t.Error))
		}

		return nil, nil
	})
	return e
}

// List logically deleted files
func ListLDelFile(ec common.ExecContext, idOffset int64, limit int) ([]File, error) {
	var l []File = []File{}

	t := mysql.GetMySql().
		Raw("select * from file where id > ? and status = ? limit ?", idOffset, STATUS_LOGIC_DEL, limit).
		Scan(&l)
	if t.Error != nil {
		return nil, fmt.Errorf("failed to list logically deleted files, %v", t.Error)
	}

	return l, nil
}

// Mark file as physically deleted by changing it's status
func PhyDelFile(ec common.ExecContext, fileId string, op PDelFileOp) error {
	fileId = strings.TrimSpace(fileId)
	if fileId == "" {
		return common.NewWebErrCode(INVALID_REQUEST, "fileId is required")
	}

	_, e := red.RLockRun(ec, FileLockKey(fileId), func() (any, error) {
		f, er := FindFile(fileId)
		if er != nil {
			return nil, common.NewWebErrCode(UNKNOWN_ERROR, er.Error())
		}

		if f.IsZero() {
			return nil, ErrFileDeleted
		}

		if !f.IsLogiDeleted() {
			return nil, nil
		}

		if ed := op.delete(ec, fileId); ed != nil {
			return nil, ed
		}

		t := mysql.GetMySql().
			Exec("update file set status = ?, phy_del_time = ? where file_id = ?", STATUS_PHYSIC_DEL, time.Now(), fileId)
		if t.Error != nil {
			return nil, common.NewWebErrCode(UNKNOWN_ERROR, fmt.Sprintf("Failed to update file, %v", t.Error))
		}

		return nil, nil
	})
	return e
}

// Concatenate file's redis lock key
func FileLockKey(fileId string) string {
	return "fstore:file:" + fileId
}

func SanitizeStorage(c common.ExecContext) error {
	dirPath := common.GetPropStr(PROP_STORAGE_DIR)
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

	c.Log.Infof("Found %v files", len(files))
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
		to, e := GenTrashPath(c, fileId)
		if e != nil {
			return fmt.Errorf("failed to GenTrashPath, %v", e)
		}

		if common.GetPropBool(PROP_SANITIZE_STORAGE_TASK_DRY_RUN) { // dry-run
			c.Log.Infof("Sanitizing storage, (dry-run) will rename file from %s to %s", frm, to)
		} else {
			if e := os.Rename(frm, to); e != nil {
				if os.IsNotExist(e) {
					c.Log.Infof("File has been deleted, file: %s", frm)
					continue
				}
				c.Log.Errorf("Sanitizing storage, failed to rename file from %s to %s, %v", frm, to, e)
			}
			c.Log.Infof("Sanitizing storage, renamed file from %s to %s", frm, to)
		}
	}
	return nil
}
