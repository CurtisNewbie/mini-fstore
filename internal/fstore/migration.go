package fstore

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"gorm.io/gorm"
)

const (
	FILE_ID_COL        = "fstore_file_id"
	FILE_INFO_MIGR_SQL = "ALTER TABLE file_info ADD COLUMN fstore_file_id VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'mini-fstore file id';"
)

type FileInfo struct {
	Id         int64
	Uuid       string
	Name       string
	UploaderId int
	FsGroupId  int64
}

type TableCol struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default string
	Extra   string
}

func init() {
	miso.SetDefProp(PropMigrFileServerDryRun, true)
}

func TryMigrateFileServer(rail miso.Rail) error {
	if !miso.GetPropBool(PropMigrFileServerEnabled) {
		return nil
	}
	return MigrateFileServer(rail)
}

/*
Migrate from file-server

Files must be copied to mini-fstore's machine beforehand (at least somewhere mini-fstore can access).

The location of these files must be specified in property: 'fstore.migr.file-miso.storage'.
*/
func MigrateFileServer(rail miso.Rail) error {
	// initialize mysql connection egaerly for file-server migration
	if e := miso.InitMySQLFromProp(rail); e != nil {
		rail.Fatalf("Failed to establish connection to MySQL, %v", e)
	}

	now := time.Now()
	defer miso.TimeOp(rail, now, "File-Server Migration")

	dryrun := miso.GetPropBool(PropMigrFileServerDryRun)
	rail.Infof("Preparing to migrate from file-server, dry-run: %v", dryrun)

	db := miso.GetPropStr(PropMigrFileServerMySQLDatabase)
	host := miso.GetPropStr(PropMigrFileServerMySQLHost)
	port := miso.GetPropInt(PropMigrFileServerMySQLPort)

	rail.Infof("Connecting to file-server's database (%s:%d/%s)", host, port, db)
	param := miso.MySQLConnParam{
		User:      miso.GetPropStr(PropMigrFileServerMySQLUser),
		Password:  miso.GetPropStr(PropMigrFileServerMySQLPwd),
		Schema:    db,
		Host:      host,
		Port:      port,
		ConnParam: miso.GetPropStr(miso.PropMySQLConnParam),
	}
	fsconn, en := miso.NewMySQLConn(rail, param)
	if en != nil {
		return fmt.Errorf("failed to connect to (%s:%d/%s), %v", host, port, db, en)
	}
	rail.Infof("File-server's database (%s:%d/%s) connected", host, port, db)
	defer func() {
		d, err := fsconn.DB()
		if err != nil {
			return
		}
		d.Close()
		rail.Infof("File-server's database (%s:%d/%s) disconnected", host, port, db)
	}()

	if !miso.IsProdMode() {
		fsconn = fsconn.Debug()
	}

	// check if file_info has been altered to add the new column for file_id
	// the new column is required, and is taken care of by mini-fstore during the migration
	var tx *gorm.DB
	var fileInfoCols []TableCol
	tx = fsconn.Raw("desc file_info").Scan(&fileInfoCols)
	if tx.Error != nil {
		return fmt.Errorf("failed to describe table file_info, %v", tx.Error)
	}
	rail.Debugf("Desc file_info: %+v", fileInfoCols)

	hasFstoreFileId := false
	for _, col := range fileInfoCols {
		if col.Field == FILE_ID_COL && strings.HasPrefix(strings.ToLower(col.Type), "varchar") {
			hasFstoreFileId = true
			break
		}
	}
	if !hasFstoreFileId {
		rail.Errorf("Table file_info doesn't have column %v, please run the following SQL before migration\n\n%s\n", FILE_ID_COL, FILE_INFO_MIGR_SQL)
		return fmt.Errorf("table file_info doesn't have column %v", FILE_ID_COL)
	}

	// where the file-server files are located at, these file must be copied to mini-fstore's machine manually before the migration
	basePath := miso.GetPropStr(PropMigrFileServerStorage)
	if basePath == "" {
		return fmt.Errorf("please specify basePath using propery: '%s'", PropMigrFileServerStorage)
	}

	// fetch file_info list, and migrate them one by one
	rail.Infof("Start migrating file-server's file_info to mini-fstore's file, file-server base path: '%v'", basePath)
	var idOffset int64 = 0

	for {
		var fileInfos []FileInfo
		tx = fsconn.
			Raw(fmt.Sprintf(`select id, uuid, name, uploader_id, fs_group_id from file_info where is_logic_deleted = 0 and file_type = 'FILE' and upload_time < ? and id > ? and %s = "" limit 1000`, FILE_ID_COL), now, idOffset).
			Scan(&fileInfos)
		if tx.Error != nil {
			return fmt.Errorf("failed to list file_info, %v", tx.Error)
		}

		if len(fileInfos) < 1 {
			break
		}

		for _, f := range fileInfos {
			// migrate each one of them
			if em := migrateFileInfo(rail, fsconn, f, basePath, dryrun); em != nil {
				return fmt.Errorf("failed to migrate file_info: %+v, %v", f, em)
			}
		}

		idOffset = fileInfos[len(fileInfos)-1].Id
	}
	rail.Info("Finished migrating file-server's file_info to mini-fstore's file")

	return nil
}

// Migrate FileInfo to mini-fstore
func migrateFileInfo(rail miso.Rail, fsconn *gorm.DB, fi FileInfo, basePath string, dryrun bool) error {
	path := fileServerPath(fi, basePath)

	f, eo := os.Open(path)
	if eo != nil {
		if os.IsNotExist(eo) {
			rail.Warnf("File not exists, skipped, uuid: %v, path: %v, %v", fi.Uuid, path, eo)
			return nil
		}
		return eo
	}
	defer f.Close()

	if dryrun {
		storage := miso.GetPropStr(PropStorageDir)
		rail.Infof("Will copy file '%s' to '%s'", path, storage)
		return nil
	}

	fileId, eu := UploadFile(rail, f, fi.Name)
	if eu != nil {
		return eu
	}

	rail.Infof("Uploaded file '%s' to mini-fstore, updating file_info, uuid: %s, fileId: %s", path, fi.Uuid, fileId)

	tx := fsconn.Exec(fmt.Sprintf("update file_info set %s = ? where id = ?", FILE_ID_COL), fileId, fi.Id)
	return tx.Error
}

// Build path to a file-server file
func fileServerPath(f FileInfo, basePath string) string {
	sep := string(os.PathSeparator)
	return basePath + sep + strconv.Itoa(f.UploaderId) + sep + f.Uuid
}
