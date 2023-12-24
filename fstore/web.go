package fstore

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/goauth"
	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

const (
	PropEnableFstoreBackup = "fstore.backup.enabled"
	ResCodeFstoreUpload    = "fstore-upload"

	ModeCluster = "cluster" // server mode - cluster (default)
	ModeProxy   = "proxy"   // server mode - proxy
	ModeNode    = "node"    // server mode - node
)

type FileInfoReq struct {
	FileId       string `form:"fileId"`
	UploadFileId string `form:"uploadFileId"`
}

func init() {
	miso.SetDefProp(PropEnableFstoreBackup, false)
	miso.SetDefProp(PROP_SERVER_MODE, ModeCluster)
	miso.SetDefProp(PROP_MIGR_FILE_SERVER_ENABLED, false)
}

/*
Parse ByteRange Request.

e.g., bytes = 123-124
*/
func parseByteRangeRequest(c *gin.Context) ByteRange {
	ranges := c.Request.Header["Range"] // e.g., Range: bytes = 1-2
	if len(ranges) < 1 {
		return ByteRange{Start: 0, End: math.MaxInt64}
	}
	return parseByteRangeHeader(ranges[0])
}

/*
Parse ByteRange Header.

e.g., bytes=123-124
*/
func parseByteRangeHeader(rangeHeader string) ByteRange {
	var start int64 = 0
	var end int64 = math.MaxInt64

	eqSplit := strings.Split(rangeHeader, "=") // split by '='
	if len(eqSplit) <= 1 {                     // 'bytes=' or '=1-2', both are illegal
		return ByteRange{Start: start, End: end}
	}

	brr := []rune(strings.TrimSpace(eqSplit[1]))
	if len(brr) < 1 { // empty byte ranges, illegal
		return ByteRange{Start: start, End: end}
	}

	dash := -1
	for i := 0; i < len(brr); i++ { // try to find the first '-'
		if brr[i] == '-' {
			dash = i
			break
		}
	}

	if dash == 0 { // the '-2' case, only the end is specified, start will still be 0
		if v, e := strconv.ParseInt(string(brr[dash+1:]), 10, 0); e == nil {
			end = v
		}
	} else if dash == len(brr)-1 { // the '1-' case, only the start is specified, end will be MaxInt64
		if v, e := strconv.ParseInt(string(brr[:dash]), 10, 0); e == nil {
			start = v
		}

	} else if dash < 0 { // the '-' case, both start and end are not specified
		// do nothing

	} else { // '1-2' normal case
		if v, e := strconv.ParseInt(string(brr[:dash]), 10, 0); e == nil {
			start = v
		}

		if v, e := strconv.ParseInt(string(brr[dash+1:]), 10, 0); e == nil {
			end = v
		}
	}
	return ByteRange{Start: start, End: end}
}

func prepareCluster(rail miso.Rail) error {
	miso.BaseRoute("/file").
		Group(
			miso.RawGet("/stream", StreamFileEp).Extra(goauth.Public("Fstore Media Streaming")),
			miso.RawGet("/raw", DownloadFileEp).Extra(goauth.Public("Fstore Raw File Download")),
			miso.Put("", UploadFileEp).Extra(goauth.Protected("Fstore File Upload", ResCodeFstoreUpload)),
			miso.IGet("/info", GetFileInfoEp),
			miso.IGet("/key", GenFileKeyEp),
			miso.IDelete("", DeleteFileEp),
		)

	// endpoints for file backup
	if miso.GetPropBool(PropEnableFstoreBackup) && miso.GetPropStr(PropBackupAuthSecret) != "" {
		rail.Infof("Enabled file backup endpoints")
		miso.BaseRoute("/backup").
			Group(
				miso.IPost("/file/list", BackupListFilesEp).Extra(goauth.Public("Backup tool list files")),
				miso.RawGet("/file/raw", BackupDownFileEp).Extra(goauth.Public("Backup tool download file")),
			)
	}

	// report paths, resources to goauth if enabled
	goauth.ReportResourcesOnBootstrapped(rail, []goauth.AddResourceReq{
		{
			Name: "Fstore File Upload",
			Code: ResCodeFstoreUpload,
		},
	})
	goauth.ReportPathsOnBootstrapped(rail)

	// register tasks
	if e := miso.ScheduleDistributedTask(miso.Job{
		Name:            "PhyDelFileTask",
		Run:             BatchPhyDelFiles,
		Cron:            "0 */1 * * *",
		CronWithSeconds: false,
	}); e != nil {
		return e
	}
	if e := miso.ScheduleDistributedTask(miso.Job{
		Name:            "SanitizeStorageTask",
		Run:             SanitizeStorage,
		Cron:            "0 */6 * * *",
		CronWithSeconds: false,
	}); e != nil {
		return e
	}

	return nil
}

func startMigration(rail miso.Rail) error {
	if !miso.GetPropBool(PROP_MIGR_FILE_SERVER_ENABLED) {
		return nil
	}
	return MigrateFileServer(rail)
}

func PrepareServer(rail miso.Rail) error {
	common.LoadBuiltinPropagationKeys()

	// migrate if necessary, server is not bootstrapped yet while we are migrating
	em := startMigration(rail)
	if em != nil {
		return fmt.Errorf("failed to migrate, %v", em)
	}

	// only supports cluster mode for now
	return prepareCluster(rail)
}

func BackupListFilesEp(c *gin.Context, rail miso.Rail, req ListBackupFileReq) (any, error) {
	auth := getAuthorization(c)
	if err := CheckBackupAuth(rail, auth); err != nil {
		return nil, err
	}

	rail.Infof("Backup tool listing files %+v", req)
	return ListBackupFiles(rail, miso.GetMySQL(), req)
}

func BackupDownFileEp(c *gin.Context, rail miso.Rail) {

	auth := getAuthorization(c)
	if err := CheckBackupAuth(rail, auth); err != nil {
		rail.Infof("CheckBackupAuth failed, %v", err)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	fileId := strings.TrimSpace(c.Query("fileId"))
	rail.Infof("Backup tool download file, fileId: %v", fileId)

	if e := DownloadFile(rail, c, fileId); e != nil {
		rail.Errorf("Download file failed, %v", e)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
}

// mark file deleted
func DeleteFileEp(c *gin.Context, rail miso.Rail, req DeleteFileReq) (any, error) {
	fileId := strings.TrimSpace(req.FileId)
	if fileId == "" {
		return nil, miso.NewErrCode(FILE_NOT_FOUND, "File is not found")
	}
	return nil, LDelFile(rail, fileId)
}

// generate random file key for downloading the file
func GenFileKeyEp(c *gin.Context, rail miso.Rail, req DownloadFileReq) (any, error) {
	timer := miso.NewPromTimer("mini_fstore_generate_file_key")
	defer timer.ObserveDuration()

	fileId := strings.TrimSpace(req.FileId)
	if fileId == "" {
		return nil, miso.NewErrCode(FILE_NOT_FOUND, "File is not found")
	}

	filename := req.Filename
	unescaped, err := url.QueryUnescape(req.Filename)
	if err == nil {
		filename = unescaped
	}
	filename = strings.TrimSpace(filename)

	k, re := RandFileKey(rail, filename, fileId)
	rail.Infof("Generated random key %v for fileId %v (%v)", k, fileId, filename)
	return k, re
}

// Get file's info
func GetFileInfoEp(c *gin.Context, rail miso.Rail, req FileInfoReq) (any, error) {
	// fake fileId for uploaded file
	if req.UploadFileId != "" {
		rcmd := miso.GetRedis().Get("mini-fstore:upload:fileId:" + req.UploadFileId)
		if rcmd.Err() != nil {
			if errors.Is(rcmd.Err(), redis.Nil) { // invalid fileId, or the uploadFileId has expired
				return nil, miso.NewErrCode(FILE_NOT_FOUND, FILE_NOT_FOUND)
			}
			return nil, rcmd.Err()
		}
		req.FileId = rcmd.Val() // the cached fileId, the real one
	}

	// using real fileId
	if req.FileId == "" {
		return nil, miso.NewErrCode(FILE_NOT_FOUND, FILE_NOT_FOUND)
	}

	f, ef := FindFile(req.FileId)
	if ef != nil {
		return nil, ef
	}
	if f.IsZero() {
		return f, miso.NewErrCode(FILE_NOT_FOUND, "File is not found")
	}
	return f, nil
}

func UploadFileEp(c *gin.Context, rail miso.Rail) (any, error) {
	fname := strings.TrimSpace(c.GetHeader("filename"))
	if fname == "" {
		return nil, miso.NewErrCode(INVALID_REQUEST, "filename is required")
	}

	fileId, e := UploadFile(rail, c.Request.Body, fname)
	if e != nil {
		return nil, e
	}

	// generate a random file key for the backend server to retrieve the
	// actual fileId later (this is to prevent user guessing others files' fileId,
	// the fileId should be used internally within the system)
	fakeFileId := miso.ERand(40)

	cmd := miso.GetRedis().Set("mini-fstore:upload:fileId:"+fakeFileId, fileId, 6*time.Hour)
	if cmd.Err() != nil {
		return nil, fmt.Errorf("failed to cache the generated fake fileId, %v", e)
	}
	rail.Infof("Generated fake fileId '%v' for '%v'", fakeFileId, fileId)

	return fakeFileId, nil
}

// Download file
func DownloadFileEp(c *gin.Context, rail miso.Rail) {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.AbortWithStatus(404)
		return
	}

	if e := DownloadFileKey(rail, c, key); e != nil {
		rail.Warnf("Failed to download by fileKey, %v", e)
		c.AbortWithStatus(404)
		return
	}
}

// Stream file (support byte-range requests)
func StreamFileEp(c *gin.Context, rail miso.Rail) {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.AbortWithStatus(404)
		return
	}

	if e := StreamFileKey(rail, c, key, parseByteRangeRequest(c)); e != nil {
		rail.Warnf("Failed to stream by fileKey, %v", e)
		c.AbortWithStatus(404)
		return
	}
}
