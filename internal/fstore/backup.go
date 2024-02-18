package fstore

import (
	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	authorization        = "Authorization"
	PropBackupAuthSecret = "fstore.backup.secret"
)

var (
	ErrInvalidAuth = miso.NewErrf("Invalid authorization").WithCode(api.InvalidAuthorization)
)

type BackupFileInf struct {
	Id     int64
	FileId string
	Name   string
	Status string
	Size   int64
	Md5    string
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
		return ListBackupFileResp{}, ErrUnknownError.WithInternalMsg("Failed to list back up files, req %+v, %v", req, err)
	}
	if files == nil {
		files = []BackupFileInf{}
	}
	return ListBackupFileResp{Files: files}, nil
}

func CheckBackupAuth(rail miso.Rail, auth string) error {
	rail.Debugf("Checking backup auth, auth: %v", auth)
	if auth == "" {
		return ErrInvalidAuth.WithInternalMsg("auth is empty")
	}
	secret := miso.GetPropStr(PropBackupAuthSecret)
	if secret != auth {
		return ErrInvalidAuth.WithInternalMsg("secret != auth")
	}
	return nil
}

func getAuthorization(c *gin.Context) string {
	return c.GetHeader(authorization)
}
