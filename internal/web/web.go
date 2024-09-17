package web

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/mini-fstore/internal/config"
	"github.com/curtisnewbie/mini-fstore/internal/fstore"
	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/middleware/redis"
	"github.com/curtisnewbie/miso/middleware/user-vault/auth"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
)

var (
	genFileKeyHisto = miso.NewPromHisto("mini_fstore_generate_file_key_duration")
)

const (
	authorization = "Authorization"
)

func RegisterRoutes(rail miso.Rail) error {

	miso.RawGet("/file/stream", TempKeyStreamFileEp).
		Desc(`
			Media streaming using temporary file key, the file_key's ttl is extended with each subsequent request.
			This endpoint is expected to be accessible publicly without authorization, since a temporary file_key
			is generated and used.
		`).
		Public().
		DocQueryParam("key", "temporary file key")

	miso.RawGet("/file/raw", TempKeyDownloadFileEp).
		Desc(`
			Download file using temporary file key. This endpoint is expected to be accessible publicly without
			authorization, since a temporary file_key is generated and used.
		`).
		Public().
		DocQueryParam("key", "temporary file key")

	miso.Put("/file", UploadFileEp).
		Desc("Upload file. A temporary file_id is returned, which should be used to exchange the real file_id").
		Resource(ResCodeFstoreUpload).
		DocHeader("filename", "name of the uploaded file")

	miso.IGet("/file/info", GetFileInfoEp).
		Desc("Fetch file info")

	miso.IGet("/file/key", GenFileKeyEp).
		Desc(`
			Generate temporary file key for downloading and streaming. This endpoint is expected to be called
			internally by another backend service that validates the ownership of the file properly.
		`)

	miso.RawGet("/file/direct", DirectDownloadFileEp).
		Desc(`
			Download files directly using file_id. This endpoint is expected to be protected and only used
			internally by another backend service. Users can eaily steal others file_id and attempt to
			download the file, so it's better not be exposed to the end users.
		`).
		DocQueryParam("fileId", "actual file_id of the file record")

	miso.IDelete("/file", DeleteFileEp).
		Desc("Mark file as deleted.")

	miso.IPost("/file/unzip", UnzipFileEp).
		Desc("Unzip archive, upload all the zip entries, and reply the final results back to the caller asynchronously")

	// endpoints for file backup
	if miso.GetPropBool(config.PropEnableFstoreBackup) && miso.GetPropStr(config.PropBackupAuthSecret) != "" {
		rail.Infof("Enabled file backup endpoints")

		miso.IPost("/backup/file/list", BackupListFilesEp).
			Desc("Backup tool list files").
			Public().
			DocHeader("Authorization", "Basic Authorization")

		miso.RawGet("/backup/file/raw", BackupDownFileEp).
			Desc("Backup tool download file").
			Public().
			DocHeader("Authorization", "Basic Authorization").
			DocQueryParam("fileId", "actual file_id of the file record")
	}

	// curl -X POST http://localhost:8084/maintenance/remove-deleted
	miso.Post("/maintenance/remove-deleted", RemoveDeletedFilesEp).
		Desc("Remove files that are logically deleted and not linked (symbolically)")

	// curl -X POST http://localhost:8084/maintenance/sanitize-storage
	miso.Post("/maintenance/sanitize-storage", SanitizeStorageEp).
		Desc("Sanitize storage, remove files in storage directory that don't exist in database")

	// curl -X POST http://localhost:8084/maintenance/compute-checksum
	miso.Post("/maintenance/compute-checksum", ComputeChecksumEp).
		Desc("Compute files' checksum if absent")

	auth.ExposeResourceInfo([]auth.Resource{
		{Name: "Fstore File Upload", Code: ResCodeFstoreUpload},
	})

	return nil
}

func BackupListFilesEp(inb *miso.Inbound, req fstore.ListBackupFileReq) (fstore.ListBackupFileResp, error) {
	rail := inb.Rail()
	_, r := inb.Unwrap()
	auth := getAuthorization(r)
	if err := fstore.CheckBackupAuth(rail, auth); err != nil {
		return fstore.ListBackupFileResp{}, err
	}

	rail.Infof("Backup tool listing files %+v", req)
	return fstore.ListBackupFiles(rail, mysql.GetMySQL(), req)
}

func BackupDownFileEp(inb *miso.Inbound) {
	rail := inb.Rail()
	w, r := inb.Unwrap()
	auth := getAuthorization(r)
	if err := fstore.CheckBackupAuth(rail, auth); err != nil {
		rail.Infof("CheckBackupAuth failed, %v", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	fileId := strings.TrimSpace(r.URL.Query().Get("fileId"))
	rail.Infof("Backup tool download file, fileId: %v", fileId)

	if e := fstore.DownloadFile(rail, w, fileId); e != nil {
		rail.Errorf("Download file failed, %v", e)
		w.WriteHeader(http.StatusForbidden)
		return
	}
}

type DeleteFileReq struct {
	FileId string `form:"fileId" valid:"notEmpty" desc:"actual file_id of the file record"`
}

// mark file deleted
func DeleteFileEp(inb *miso.Inbound, req DeleteFileReq) (any, error) {
	rail := inb.Rail()
	fileId := strings.TrimSpace(req.FileId)
	if fileId == "" {
		return nil, fstore.ErrFileNotFound
	}
	return nil, fstore.LDelFile(rail, mysql.GetMySQL(), fileId)
}

type DownloadFileReq struct {
	FileId   string `form:"fileId" desc:"actual file_id of the file record"`
	Filename string `form:"filename" desc:"the name that will be used when downloading the file"`
}

// generate random file key for downloading the file
func GenFileKeyEp(inb *miso.Inbound, req DownloadFileReq) (string, error) {
	rail := inb.Rail()
	timer := miso.NewHistTimer(genFileKeyHisto)
	defer timer.ObserveDuration()

	fileId := strings.TrimSpace(req.FileId)
	if fileId == "" {
		return "", fstore.ErrFileNotFound
	}

	filename := req.Filename
	unescaped, err := url.QueryUnescape(req.Filename)
	if err == nil {
		filename = unescaped
	}
	filename = strings.TrimSpace(filename)

	k, re := fstore.RandFileKey(rail, filename, fileId)
	rail.Infof("Generated random key %s for fileId %s (using filename: '%s')", k, fileId, filename)
	return k, re
}

type FileInfoReq struct {
	FileId       string `form:"fileId" desc:"actual file_id of the file record"`
	UploadFileId string `form:"uploadFileId" desc:"temporary file_id returned when uploading files"`
}

// Get file's info
func GetFileInfoEp(inb *miso.Inbound, req FileInfoReq) (api.FstoreFile, error) {
	// fake fileId for uploaded file
	if req.UploadFileId != "" {
		rcmd := redis.GetRedis().Get("mini-fstore:upload:fileId:" + req.UploadFileId)
		if rcmd.Err() != nil {
			if redis.IsNil(rcmd.Err()) {
				// invalid fileId, or the uploadFileId has expired
				return api.FstoreFile{}, fstore.ErrFileNotFound
			}
			return api.FstoreFile{}, rcmd.Err()
		}
		req.FileId = rcmd.Val() // the cached fileId, the real one
	}

	// using real fileId
	if req.FileId == "" {
		return api.FstoreFile{}, fstore.ErrFileNotFound
	}

	f, ef := fstore.FindFile(mysql.GetMySQL(), req.FileId)
	if ef != nil {
		return api.FstoreFile{}, ef
	}
	if f.IsZero() {
		return api.FstoreFile{}, fstore.ErrFileNotFound
	}
	return api.FstoreFile{
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

func UploadFileEp(inb *miso.Inbound) (string, error) {
	rail := inb.Rail()
	_, r := inb.Unwrap()
	fname := strings.TrimSpace(r.Header.Get("filename"))
	if fname == "" {
		return "", fstore.ErrFilenameRequired
	}
	if n, err := url.QueryUnescape(fname); err == nil {
		fname = n
	}

	fileId, e := fstore.UploadFile(rail, r.Body, fname)
	if e != nil {
		return "", e
	}

	// generate a random file key for the backend server to retrieve the
	// actual fileId later (this is to prevent user guessing others files' fileId,
	// the fileId should be used internally within the system)
	tempFileId := util.ERand(40)

	cmd := redis.GetRedis().Set("mini-fstore:upload:fileId:"+tempFileId, fileId, 6*time.Hour)
	if cmd.Err() != nil {
		return "", fmt.Errorf("failed to cache the generated fake fileId, %v", e)
	}
	rail.Infof("Generated fake fileId '%v' for '%v'", tempFileId, fileId)

	return tempFileId, nil
}

// Download file
func TempKeyDownloadFileEp(inb *miso.Inbound) {
	rail := inb.Rail()
	w, r := inb.Unwrap()
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		w.WriteHeader(404)
		return
	}

	if e := fstore.DownloadFileKey(rail, w, key); e != nil {
		rail.Warnf("Failed to download by fileKey, %v", e)
		w.WriteHeader(404)
		return
	}
}

// Stream file (support byte-range requests)
func TempKeyStreamFileEp(inb *miso.Inbound) {
	rail := inb.Rail()
	w, r := inb.Unwrap()
	query := r.URL.Query()
	key := strings.TrimSpace(query.Get("key"))
	if key == "" {
		w.WriteHeader(404)
		return
	}

	if e := fstore.StreamFileKey(rail, w, key, parseByteRangeRequest(r)); e != nil {
		rail.Warnf("Failed to stream by fileKey, %v", e)
		w.WriteHeader(404)
		return
	}
}

func UnzipFileEp(inb *miso.Inbound, req api.UnzipFileReq) (any, error) {
	rail := inb.Rail()
	return nil, fstore.TriggerUnzipFilePipeline(rail, mysql.GetMySQL(), req)
}

/*
Parse ByteRange Request.

e.g., bytes = 123-124
*/
func parseByteRangeRequest(r *http.Request) fstore.ByteRange {
	headers := r.Header
	rg := headers.Get("Range") // e.g., Range: bytes = 1-2
	if rg == "" {
		return fstore.ByteRange{Start: 0, End: math.MaxInt64}
	}
	return parseByteRangeHeader(rg)
}

/*
Parse ByteRange Header.

e.g., bytes=123-124
*/
func parseByteRangeHeader(rangeHeader string) fstore.ByteRange {
	var start int64 = 0
	var end int64 = math.MaxInt64

	eqSplit := strings.Split(rangeHeader, "=") // split by '='
	if len(eqSplit) <= 1 {                     // 'bytes=' or '=1-2', both are illegal
		return fstore.ByteRange{Start: start, End: end}
	}

	brr := []rune(strings.TrimSpace(eqSplit[1]))
	if len(brr) < 1 { // empty byte ranges, illegal
		return fstore.ByteRange{Start: start, End: end}
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
	return fstore.ByteRange{Start: start, End: end}
}

func RemoveDeletedFilesEp(inb *miso.Inbound) (any, error) {
	rail := inb.Rail()
	return nil, fstore.RemoveDeletedFiles(rail, mysql.GetMySQL())
}

func SanitizeStorageEp(inb *miso.Inbound) (any, error) {
	rail := inb.Rail()
	return nil, fstore.SanitizeStorage(rail)
}

func DirectDownloadFileEp(inb *miso.Inbound) {
	rail := inb.Rail()
	w, r := inb.Unwrap()
	fileId := r.URL.Query().Get("fileId")
	if fileId == "" {
		rail.Warnf("missing fileId")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if e := fstore.DownloadFile(rail, w, fileId); e != nil {
		rail.Warnf("Failed to DownloadFile using fileId: %v, %v", fileId, e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func getAuthorization(r *http.Request) string {
	return r.Header.Get(authorization)
}

func ComputeChecksumEp(inb *miso.Inbound) (any, error) {
	rail := inb.Rail()
	return nil, fstore.ComputeFilesChecksum(rail, mysql.GetMySQL())
}
