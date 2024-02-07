package fstore

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	authorization        = "Authorization"
	PropBackupAuthSecret = "fstore.backup.secret"
)

var (
	errInvalidAuth = miso.NewErrf("Invalid authorization")
)

type BackupFileInf struct {
	Id     int64
	FileId string
	Name   string
	Status string
	Size   int64
	Md5    string
}

type ListBackupFileReq struct {
	Limit    int64
	IdOffset int
}

type ListBackupFileResp struct {
	Files []BackupFileInf
}

func ListBackupFiles(rail miso.Rail, tx *gorm.DB, req ListBackupFileReq) (ListBackupFileResp, error) {
	var files []BackupFileInf
	err := tx.
		Table("file").
		Select("id, file_id, name, status, size, md5").
		Where("id > ?", req.IdOffset).
		Order("id ASC").
		Limit(int(req.Limit)).
		Scan(&files).
		Error
	if err != nil {
		return ListBackupFileResp{}, miso.NewErrf("Unknown error").WithInternalMsg("Failed to list back up files, req %+v, %v", req, err)
	}
	if files == nil {
		files = []BackupFileInf{}
	}
	return ListBackupFileResp{Files: files}, nil
}

func CheckBackupAuth(rail miso.Rail, auth string) error {
	rail.Debugf("Checking backup auth, auth: %v", auth)
	if auth == "" {
		return errInvalidAuth
	}
	secret := miso.GetPropStr(PropBackupAuthSecret)
	if secret != auth {
		return errInvalidAuth
	}
	return nil
}

func getAuthorization(c *gin.Context) string {
	return c.GetHeader(authorization)
}
