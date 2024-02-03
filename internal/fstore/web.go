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

	"github.com/curtisnewbie/gocommon/goauth"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var (
	genFileKeyHisto = miso.NewPromHisto("mini_fstore_generate_file_key_duration")
)

func registerRoutes(rail miso.Rail) error {
	miso.BaseRoute("/file").
		Group(
			miso.RawGet("/stream", StreamFileEp).Desc("Fstore Media Streaming").Public(),
			miso.RawGet("/raw", DownloadFileEp).Desc("Fstore Raw File Download").Public(),
			miso.Put("", UploadFileEp).Desc("Fstore File Upload").Resource(ResCodeFstoreUpload),
			miso.IGet("/info", GetFileInfoEp),
			miso.IGet("/key", GenFileKeyEp),
			miso.IDelete("", DeleteFileEp),
			miso.IPost("/unzip", UnzipFileEp),
		)

	// endpoints for file backup
	if miso.GetPropBool(PropEnableFstoreBackup) && miso.GetPropStr(PropBackupAuthSecret) != "" {
		rail.Infof("Enabled file backup endpoints")
		miso.BaseRoute("/backup").
			Group(
				miso.IPost("/file/list", BackupListFilesEp).Desc("Backup tool list files").Public(),
				miso.RawGet("/file/raw", BackupDownFileEp).Desc("Backup tool download file").Public(),
			)
	}

	// curl -X POST http://localhost:8084/maintenance/remove-deleted
	miso.BaseRoute("/maintenance").
		Group(
			// remove files that are logically deleted and not linked (symbolically)
			miso.Post("/remove-deleted", func(c *gin.Context, rail miso.Rail) (any, error) {
				SanitizeDeletedFiles(rail, miso.GetMySQL())
				return nil, nil
			}),
		)

	// report paths, resources to goauth if enabled
	goauth.ReportOnBoostrapped(rail, []goauth.AddResourceReq{
		{
			Name: "Fstore File Upload",
			Code: ResCodeFstoreUpload,
		},
	})

	// register tasks
	if e := miso.ScheduleDistributedTask(miso.Job{
		Name:            "SanitizeStorageTask",
		Run:             SanitizeStorage,
		Cron:            "0 */12 * * *",
		CronWithSeconds: false,
	}); e != nil {
		return e
	}

	return nil
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

type DeleteFileReq struct {
	FileId string `form:"fileId" valid:"notEmpty"`
}

// mark file deleted
func DeleteFileEp(c *gin.Context, rail miso.Rail, req DeleteFileReq) (any, error) {
	fileId := strings.TrimSpace(req.FileId)
	if fileId == "" {
		return nil, miso.NewErrCode(api.FileNotFound, "File is not found")
	}
	return nil, LDelFile(rail, miso.GetMySQL(), fileId)
}

type DownloadFileReq struct {
	FileId   string `form:"fileId"`
	Filename string `form:"filename"`
}

// generate random file key for downloading the file
func GenFileKeyEp(c *gin.Context, rail miso.Rail, req DownloadFileReq) (any, error) {
	timer := miso.NewHistTimer(genFileKeyHisto)
	defer timer.ObserveDuration()

	fileId := strings.TrimSpace(req.FileId)
	if fileId == "" {
		return nil, miso.NewErrCode(api.FileNotFound, "File is not found")
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

type FileInfoReq struct {
	FileId       string `form:"fileId"`
	UploadFileId string `form:"uploadFileId"`
}

// Get file's info
func GetFileInfoEp(c *gin.Context, rail miso.Rail, req FileInfoReq) (any, error) {
	// fake fileId for uploaded file
	if req.UploadFileId != "" {
		rcmd := miso.GetRedis().Get("mini-fstore:upload:fileId:" + req.UploadFileId)
		if rcmd.Err() != nil {
			if errors.Is(rcmd.Err(), redis.Nil) { // invalid fileId, or the uploadFileId has expired
				return nil, miso.NewErrCode(api.FileNotFound, api.FileNotFound)
			}
			return nil, rcmd.Err()
		}
		req.FileId = rcmd.Val() // the cached fileId, the real one
	}

	// using real fileId
	if req.FileId == "" {
		return nil, miso.NewErrCode(api.FileNotFound, api.FileNotFound)
	}

	f, ef := FindFile(miso.GetMySQL(), req.FileId)
	if ef != nil {
		return nil, ef
	}
	if f.IsZero() {
		return nil, miso.NewErrCode(api.FileNotFound, "File is not found")
	}
	return api.FstoreFile{
		Id:         f.Id,
		FileId:     f.FileId,
		Name:       f.Name,
		Status:     f.Status,
		Size:       f.Size,
		Md5:        f.Md5,
		UplTime:    f.UplTime,
		LogDelTime: f.LogDelTime,
		PhyDelTime: f.PhyDelTime,
	}, nil
}

func UploadFileEp(c *gin.Context, rail miso.Rail) (any, error) {
	fname := strings.TrimSpace(c.GetHeader("filename"))
	if fname == "" {
		return nil, miso.NewErrCode(api.InvalidRequest, "filename is required")
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

func UnzipFileEp(c *gin.Context, rail miso.Rail, req api.UnzipFileReq) (any, error) {
	return nil, TriggerUnzipFilePipeline(rail, miso.GetMySQL(), req)
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
