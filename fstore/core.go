package fstore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/mysql"
	red "github.com/curtisnewbie/gocommon/redis"
	"github.com/go-redis/redis"
)

const (
	FILE_ID_PREFIX = "file_" // prefix of file_id

	STATUS_NORMAL     = "NORMAL"  // file.status - normal
	STATUS_LOGIC_DEL  = "LOG_DEL" // file.status - logically deletedy
	STATUS_PHYSIC_DEL = "PHY_DEL" // file.status - physically deletedy
)

var (
	_storageMkdir   int32 = 0 // did we do mkdir for storage dir; atomic value, 0 - false, 1 - true
	ErrFileNotFound       = common.NewWebErrCode(FILE_NOT_FOUND, "File is not found")
	ErrFileDeleted        = common.NewWebErrCode(FILE_DELETED, "File has been deleted already")
)

func init() {
	common.SetDefProp(PROP_STORAGE_DIR, "./storage")
	common.SetDefProp(PROP_TRASH_DIR, "./trash")
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

// Generate random file_id
func GenFileId() string {
	return common.GenIdP(FILE_ID_PREFIX)
}

// Generate file path
//
// Property `fstore.storage.dir` is used
func GenFilePath(ec common.ExecContext, fileId string) (string, error) {
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

// Create random file key for the file
func RandFileKey(ec common.ExecContext, fileId string) (string, error) {
	s, er := Rand(30)
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

	c := red.GetRedis().Set("fstore:file:key:"+s, fileId, 30*time.Minute)
	return s, c.Err()
}

// Resolve fileId for the given fileKey
func ResolveFileId(ec common.ExecContext, fileKey string) (bool, string) {
	c := red.GetRedis().Get("fstore:file:key:" + fileKey)
	if c.Err() != nil {
		if errors.Is(c.Err(), redis.Nil) {
			ec.Log.Infof("FileKey not found, %v", fileKey)
		} else {
			ec.Log.Errorf("Failed to find fileKey, %v", c.Err())
		}
		return false, ""
	}
	return true, c.Val()
}

// Download file by a generated random file key
func DownloadFileKey(ec common.ExecContext, w io.Writer, fileKey string) error {
	ok, fileId := ResolveFileId(ec, fileKey)
	if !ok {
		return ErrFileNotFound
	}
	return DownloadFile(ec, w, fileId)
}

// Download file
func DownloadFile(ec common.ExecContext, w io.Writer, fileId string) error {
	start := time.Now()
	ff, err := FindFile(fileId)
	if err != nil {
		return fmt.Errorf("failed to find file, %v", err)
	}
	if ff.IsDeleted() {
		return ErrFileDeleted
	}

	p, eg := GenFilePath(ec, fileId)
	if eg != nil {
		return fmt.Errorf("failed to generate file path, %v", eg)
	}
	ec.Log.Infof("Downloading file '%s', path: '%s'", fileId, p)

	f, eo := os.Open(p)
	if eo != nil {
		return fmt.Errorf("failed to open file, %v", eo)
	}

	l, et := io.CopyBuffer(w, f, DefBuf())
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
	target, eg := GenFilePath(ec, fileId)
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
func PhyDelFile(ec common.ExecContext, fileId string) error {
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

		if f.IsDeleted() {
			return nil, ErrFileDeleted
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
