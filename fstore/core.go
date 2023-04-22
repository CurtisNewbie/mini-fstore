package fstore

import (
	"fmt"
	"strings"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/redis"
)

const (
	FILE_ID_PREFIX = "file_" // prefix of file_id

	STATUS_NORMAL     = "NORMAL"  // file.status - normal
	STATUS_LOGIC_DEL  = "LOG_DEL" // file.status - logically deletedy
	STATUS_PHYSIC_DEL = "PHY_DEL" // file.status - physically deletedy
)

type CreateFile struct {
	FileId string
	Size   int64
	Md5    string
}

type File struct {
	Id         int64
	FileId     string
	Status     string
	Size       int64
	Md5        string
	UplTime    common.ETime
	LogDelTime common.ETime
	PhyDelTime common.ETime
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

// Create File record
func CreateFileRec(ec common.ExecContext, c CreateFile) error {
	f := File{
		FileId:  c.FileId,
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
		return File{}, fmt.Errorf("failed to select file from DB, %w", t.Error)
	}
	return f, nil
}

// Delete file logically by changing it's status
func LDelFile(ec common.ExecContext, fileId string) error {
	fileId = strings.TrimSpace(fileId)
	if fileId == "" {
		return common.NewWebErrCode(INVALID_REQUEST, "fileId is required")
	}

	_, e := redis.RLockRun(ec, FileLockKey(fileId), func() (any, error) {
		f, er := FindFile(fileId)
		if er != nil {
			return nil, common.NewWebErrCode(UNKNOWN_ERROR, er.Error())
		}

		if f.IsZero() {
			return nil, common.NewWebErrCode(FILE_NOT_FOUND, FILE_NOT_FOUND)
		}

		if f.IsDeleted() {
			return nil, common.NewWebErrCode(FILE_DELETED, "File has been deleted already")
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

	_, e := redis.RLockRun(ec, FileLockKey(fileId), func() (any, error) {
		f, er := FindFile(fileId)
		if er != nil {
			return nil, common.NewWebErrCode(UNKNOWN_ERROR, er.Error())
		}

		if f.IsZero() {
			return nil, common.NewWebErrCode(FILE_NOT_FOUND, FILE_NOT_FOUND)
		}

		if f.IsDeleted() {
			return nil, common.NewWebErrCode(FILE_DELETED, "File has been deleted already")
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
