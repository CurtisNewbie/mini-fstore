package fstore

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"reflect"
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
	miso.BaseRoute("/file").Group(

		miso.RawGet("/stream", TempKeyStreamFileEp).
			Desc(`
				Media streaming using temporary file key, the file_key's ttl is extended with each subsequent request. 
				This Endpoint is expected to be accessible publicly without authorization, since a temporary file_key 
				is generated and used.
			`).
			Public().
			DocQueryParam("key", "temporary file key"),

		miso.RawGet("/raw", TempKeyDownloadFileEp).
			Desc(`
				File download using temporary file key. This endpoint is expected to be accessible publicly without 
				authorization, since a temporary file_key is generated and used.
			`).
			Public().
			DocQueryParam("key", "temporary file key"),

		miso.Put("", UploadFileEp).
			Desc("Fstore file upload. A temporary file_id is returned, which should be used to exchange the real file_id").
			Resource(ResCodeFstoreUpload).
			DocHeader("filename", "name of the uploaded file").
			DocJsonResp(reflect.TypeOf(miso.GnResp[string]{})),

		miso.IGet("/info", GetFileInfoEp).
			Desc("Fetch file info").
			DocQueryParam("uploadFileId", "temporary file_id returned when uploading files").
			DocQueryParam("fileId", "actual file_id of the file record").
			DocJsonResp(reflect.TypeOf(miso.GnResp[api.FstoreFile]{})),

		miso.IGet("/key", GenFileKeyEp).
			Desc("Generate temporary file key for downloading and streaming. This ").
			DocQueryParam("fileId", "actual file_id of the file record").
			DocQueryParam("filename", "the name that will be used when downloading the file").
			DocJsonResp(reflect.TypeOf(miso.GnResp[string]{})),

		miso.RawGet("/direct", DirectDownloadFileEp).
			Desc(`
				Download files directly using file_id. Endpoint is expected to be protected and only used internally. 
				One may steal others file_id easily and attempt to download the file.
			`).
			DocQueryParam("fileId", "actual file_id of the file record"),

		miso.IDelete("", DeleteFileEp).
			Desc("Make file as deleted").
			DocQueryParam("fileId", "actual file_id of the file record"),

		miso.IPost("/unzip", UnzipFileEp).
			Desc("Unzip archive, upload all the zip entries, and reply the final results back to the caller asynchronously").
			DocJsonReq(reflect.TypeOf(api.UnzipFileReq{})),
	)

	// endpoints for file backup
	if miso.GetPropBool(PropEnableFstoreBackup) && miso.GetPropStr(PropBackupAuthSecret) != "" {
		rail.Infof("Enabled file backup endpoints")
		miso.BaseRoute("/backup").Group(
			miso.IPost("/file/list", BackupListFilesEp).
				Desc("Backup tool list files").
				Public().
				DocHeader("Authorization", "Basic Authorization").
				DocJsonReq(reflect.TypeOf(ListBackupFileReq{})),

			miso.RawGet("/file/raw", BackupDownFileEp).
				Desc("Backup tool download file").
				Public().
				DocHeader("Authorization", "Basic Authorization").
				DocQueryParam("fileId", "actual file_id of the file record"),
		)
	}

	miso.BaseRoute("/maintenance").Group(

		// curl -X POST http://localhost:8084/maintenance/remove-deleted
		miso.Post("/remove-deleted", RemoveDeletedFilesEp).
			Desc("Remove files that are logically deleted and not linked (symbolically)"),

		// curl -X POST http://localhost:8084/maintenance/sanitize-storage
		miso.Post("/sanitize-storage", SanitizeStorageEp).
			Desc("Sanitize storage, remove files in storage directory that don't exist in database"),
	)

	// report paths, resources to goauth if enabled
	goauth.ReportOnBoostrapped(rail, []goauth.AddResourceReq{
		{
			Name: "Fstore File Upload",
			Code: ResCodeFstoreUpload,
		},
	})

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
		return nil, miso.NewErrf("File is not found").WithCode(api.FileNotFound)
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
		return nil, miso.NewErrf("File is not found").WithCode(api.FileNotFound)
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
				return nil, miso.NewErrf("File is not found").WithCode(api.FileNotFound)
			}
			return nil, rcmd.Err()
		}
		req.FileId = rcmd.Val() // the cached fileId, the real one
	}

	// using real fileId
	if req.FileId == "" {
		return nil, miso.NewErrf("File is not found").WithCode(api.FileNotFound)
	}

	f, ef := FindFile(miso.GetMySQL(), req.FileId)
	if ef != nil {
		return nil, ef
	}
	if f.IsZero() {
		return nil, miso.NewErrf("File is not found").WithCode(api.FileNotFound)
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
		return nil, miso.NewErrf("Filename is required").WithCode(api.InvalidRequest)
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
func TempKeyDownloadFileEp(c *gin.Context, rail miso.Rail) {
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
func TempKeyStreamFileEp(c *gin.Context, rail miso.Rail) {
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

func RemoveDeletedFilesEp(c *gin.Context, rail miso.Rail) (any, error) {
	return nil, RemoveDeletedFiles(rail, miso.GetMySQL())
}

func SanitizeStorageEp(c *gin.Context, rail miso.Rail) (any, error) {
	return nil, SanitizeStorage(rail)
}

func DirectDownloadFileEp(c *gin.Context, rail miso.Rail) {
	fileId := c.Query("fileId")
	if fileId == "" {
		rail.Warnf("missing fileId")
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if e := DownloadFile(rail, c, fileId); e != nil {
		rail.Warnf("Failed to DownloadFile using fileId: %v, %v", fileId, e)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
}
