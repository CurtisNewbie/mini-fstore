package fstore

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/goauth"
	"github.com/curtisnewbie/miso/miso"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

func init() {
	miso.SetDefProp(PROP_ENABLE_GOAUTH_REPORT, false)
}

var (
	paths     = []goauth.CreatePathReq{}  // hardcoded paths for goauth
	resources = []goauth.AddResourceReq{} // hardcoded resources for goauth
)

const (
	RES_CODE_FSTORE_UPLOAD = "fstore-upload"

	MODE_CLUSTER = "cluster" // server mode - cluster (default)
	MODE_PROXY   = "proxy"   // server mode - proxy
	MODE_NODE    = "node"    // server mode - node
)

type FileInfoReq struct {
	FileId       string `form:"fileId"`
	UploadFileId string `form:"uploadFileId"`
}

func init() {
	miso.SetDefProp(PROP_ENABLE_GOAUTH_REPORT, false)
	miso.SetDefProp(PROP_SERVER_MODE, MODE_CLUSTER)
	miso.SetDefProp(PROP_MIGR_FILE_SERVER_ENABLED, false)
}

// func prepareNode(rail miso.Rail) {
// 	rail.Info("Preparing Server Using Node Mode")
// 	// TODO
// }

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
	rail.Info("Preparing Server Using Cluster Mode")

	// stream file (support byte-range requests)
	miso.RawGet("/file/stream", func(c *gin.Context, rail miso.Rail) {
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
	})

	// download file
	miso.RawGet("/file/raw", func(c *gin.Context, rail miso.Rail) {
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
	})

	// upload file
	miso.Put("/file", func(c *gin.Context, rail miso.Rail) (any, error) {
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
	})

	// get file's info
	miso.IGet("/file/info", func(c *gin.Context, rail miso.Rail, req FileInfoReq) (any, error) {
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
	})

	// generate random file key for downloading the file
	miso.IGet("/file/key", func(c *gin.Context, rail miso.Rail, req DownloadFileReq) (any, error) {
		fileId := strings.TrimSpace(req.FileId)
		if fileId == "" {
			return nil, miso.NewErrCode(FILE_NOT_FOUND, "File is not found")
		}
		filename := strings.TrimSpace(req.Filename)
		k, re := RandFileKey(rail, filename, fileId)
		rail.Infof("Generated random key %v for fileId %v (%v)", k, fileId, filename)
		return k, re
	})

	// mark file deleted
	miso.IDelete("/file", func(c *gin.Context, rail miso.Rail, req DeleteFileReq) (any, error) {
		fileId := strings.TrimSpace(req.FileId)
		if fileId == "" {
			return nil, miso.NewErrCode(FILE_NOT_FOUND, "File is not found")
		}
		return nil, LDelFile(rail, fileId)
	})

	// if goauth client is enabled, report some hardcoded paths and resources to it
	if GoAuthEnabled() {
		paths = append(paths, goauth.CreatePathReq{
			Type:   goauth.PT_PUBLIC,
			Url:    "/fstore/file/stream",
			Group:  "fstore",
			Desc:   "Fstore Media Streaming",
			Method: "GET",
		})
		paths = append(paths, goauth.CreatePathReq{
			Type:   goauth.PT_PUBLIC,
			Url:    "/fstore/file/raw",
			Group:  "fstore",
			Desc:   "Fstore Raw File Download",
			Method: "GET",
		})
		paths = append(paths, goauth.CreatePathReq{
			Type:    goauth.PT_PROTECTED,
			Url:     "/fstore/file",
			Group:   "fstore",
			Desc:    "Fstore File Upload",
			Method:  "PUT",
			ResCode: RES_CODE_FSTORE_UPLOAD,
		})

		resources = append(resources, goauth.AddResourceReq{
			Name: "Fstore File Upload",
			Code: RES_CODE_FSTORE_UPLOAD,
		})

		reportToGoAuth := func(rail miso.Rail) error {
			if e := ReportResourcesAsync(rail); e != nil {
				return fmt.Errorf("failed to report resources, %v", e)
			}
			if e := ReportPathsAsync(rail); e != nil {
				return fmt.Errorf("failed to report paths, %v", e)
			}
			return nil
		}
		miso.PostServerBootstrapped(reportToGoAuth)
	}

	// register tasks
	if e := miso.ScheduleNamedDistributedTask("0 */1 * * *", false, "PhyDelFileTask", BatchPhyDelFiles); e != nil {
		return e
	}
	if e := miso.ScheduleNamedDistributedTask("0 */6 * * *", false, "SanitizeStorageTask", SanitizeStorage); e != nil {
		return e
	}

	return nil
}

// func prepareProxy(rail miso.Rail) {
// 	rail.Info("Preparing Server Using Proxy Mode")
// 	// TODO
// }

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

// Report paths to goauth
func ReportPathsAsync(rail miso.Rail) error {
	for _, v := range paths {
		if e := goauth.AddPathAsync(rail, v); e != nil {
			return fmt.Errorf("failed to call goauth.AddPath, %v", e)
		}
	}
	return nil
}

// Check if GoAuth client is enabled
//
// This func use property 'goauth.report.enabled'
func GoAuthEnabled() bool {
	return miso.GetPropBool(PROP_ENABLE_GOAUTH_REPORT)
}

// Report resources to goauth
func ReportResourcesAsync(rail miso.Rail) error {
	for _, v := range resources {
		if e := goauth.AddResourceAsync(rail, v); e != nil {
			return fmt.Errorf("failed to call goauth.AddResource, %v", e)
		}
	}
	return nil
}
