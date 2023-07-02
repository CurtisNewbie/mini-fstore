package fstore

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/goauth"
	red "github.com/curtisnewbie/gocommon/redis"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/gocommon/task"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

func init() {
	common.SetDefProp(PROP_ENABLE_GOAUTH_REPORT, false)
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
	common.SetDefProp(PROP_ENABLE_GOAUTH_REPORT, false)
	common.SetDefProp(PROP_SERVER_MODE, MODE_CLUSTER)
	common.SetDefProp(PROP_MIGR_FILE_SERVER_ENABLED, false)
}

func prepareNode(c common.ExecContext) {
	c.Log.Info("Preparing Server Using Node Mode")
	// TODO
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

func prepareCluster(c common.ExecContext) {
	c.Log.Info("Preparing Server Using Cluster Mode")

	// stream file (support byte-range requests)
	server.RawGet("/file/stream", func(c *gin.Context, ec common.ExecContext) {
		key := strings.TrimSpace(c.Query("key"))
		if key == "" {
			c.AbortWithStatus(404)
			return
		}

		if e := StreamFileKey(ec, c, key, parseByteRangeRequest(c)); e != nil {
			ec.Log.Warnf("Failed to stream by fileKey, %v", e)
			c.AbortWithStatus(404)
			return
		}
	})

	// download file
	server.RawGet("/file/raw", func(c *gin.Context, ec common.ExecContext) {
		key := strings.TrimSpace(c.Query("key"))
		if key == "" {
			c.AbortWithStatus(404)
			return
		}

		if e := DownloadFileKey(ec, c, key); e != nil {
			ec.Log.Warnf("Failed to download by fileKey, %v", e)
			c.AbortWithStatus(404)
			return
		}
	})

	// upload file
	server.Put("/file", func(c *gin.Context, ec common.ExecContext) (any, error) {
		fname := strings.TrimSpace(c.GetHeader("filename"))
		if fname == "" {
			return nil, common.NewWebErrCode(INVALID_REQUEST, "filename is required")
		}

		fileId, e := UploadFile(ec, c.Request.Body, fname)
		if e != nil {
			return nil, e
		}

		// generate a random file key for the backend server to retrieve the
		// actual fileId later (this is to prevent user guessing others files' fileId,
		// the fileId should be used internally within the system)
		fakeFileId, e := common.ERand(40)
		if e != nil {
			return nil, fmt.Errorf("failed to generate fake fileId, %v", e)
		}

		cmd := red.GetRedis().Set("mini-fstore:upload:fileId:"+fakeFileId, fileId, 12*time.Hour)
		if cmd.Err() != nil {
			return nil, fmt.Errorf("failed to cache the generated fake fileId, %v", e)
		}
		ec.Log.Info("Generated fake fileId '%v' for '%v'", fakeFileId, fileId)

		return fakeFileId, nil
	})

	// get file's info
	server.IGet("/file/info", func(c *gin.Context, ec common.ExecContext, r FileInfoReq) (any, error) {
		// fake fileId for uploaded file
		if r.UploadFileId != "" {
			rcmd := red.GetRedis().Get("mini-fstore:upload:fileId:" + r.UploadFileId)
			if rcmd.Err() != nil {
				if errors.Is(rcmd.Err(), redis.Nil) { // invalid fileId, or the uploadFileId has expired
					return nil, common.NewWebErrCode(FILE_NOT_FOUND, FILE_NOT_FOUND)
				}
				return nil, rcmd.Err()
			}
			r.FileId = rcmd.Val() // the cached fileId, the real one
		}

		// using real fileId
		if r.FileId == "" {
			return nil, common.NewWebErrCode(FILE_NOT_FOUND, FILE_NOT_FOUND)
		}

		f, ef := FindFile(r.FileId)
		if ef != nil {
			return nil, ef
		}
		if f.IsZero() {
			return f, common.NewWebErrCode(FILE_NOT_FOUND, "File is not found")
		}
		return f, nil
	})

	// generate random file key for downloading the file
	server.Get("/file/key", func(c *gin.Context, ec common.ExecContext) (any, error) {
		fileId := strings.TrimSpace(c.Query("fileId"))
		if fileId == "" {
			return nil, common.NewWebErrCode(FILE_NOT_FOUND, "File is not found")
		}
		filename := strings.TrimSpace(c.Query("filename"))
		k, re := RandFileKey(ec, filename, fileId)
		if re == nil {
			k = url.QueryEscape(k)
		}
		return k, re
	})

	// generate random file key in batch for downloading the files
	server.IPost("/file/key/batch", func(c *gin.Context, ec common.ExecContext, req BatchGenFileKeyReq) (any, error) {
		defer common.TimeOp(ec, time.Now(), "BatchRandFileKey")
		return BatchRandFileKey(ec, req.Items)
	})

	// mark file deleted
	server.Delete("/file", func(c *gin.Context, ec common.ExecContext) (any, error) {
		fileId := strings.TrimSpace(c.Query("fileId"))
		if fileId == "" {
			return nil, common.NewWebErrCode(FILE_NOT_FOUND, "File is not found")
		}
		return nil, LDelFile(ec, fileId)
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

		reportToGoAuth := func(ec common.ExecContext) error {
			if e := ReportResources(ec); e != nil {
				return fmt.Errorf("failed to report resources, %v", e)
			}
			if e := ReportPaths(ec); e != nil {
				return fmt.Errorf("failed to report paths, %v", e)
			}
			return nil
		}
		server.PostServerBootstrapped(reportToGoAuth)
	}

	// register tasks
	task.ScheduleNamedDistributedTask("0 0 0/1 * * *", "PhyDelFileTask", func(ec common.ExecContext) error {
		return BatchPhyDelFiles(ec)
	})
	task.ScheduleNamedDistributedTask("0 0 0/6 * * *", "SanitizeStorageTask", func(ec common.ExecContext) error {
		return SanitizeStorage(ec)
	})
}

func prepareProxy(c common.ExecContext) {
	c.Log.Info("Preparing Server Using Proxy Mode")
	// TODO
}

func startMigration(c common.ExecContext) error {
	if !common.GetPropBool(PROP_MIGR_FILE_SERVER_ENABLED) {
		return nil
	}
	return MigrateFileServer(c)
}

func PrepareServer(c common.ExecContext) {
	// migrate if necessary, server is not bootstrapped yet while we are migrating
	em := startMigration(c)
	if em != nil {
		c.Log.Fatalf("Failed to migrate, %v", em)
	}

	// only supports standalone mode for now
	prepareCluster(c)
}

// Report paths to goauth
func ReportPaths(ec common.ExecContext) error {
	for _, v := range paths {
		if e := goauth.AddPath(ec.Ctx, v); e != nil {
			return fmt.Errorf("failed to call goauth.AddPath, %v", e)
		}
	}
	return nil
}

// Check if GoAuth client is enabled
//
// This func use property 'goauth.report.enabled'
func GoAuthEnabled() bool {
	return common.GetPropBool(PROP_ENABLE_GOAUTH_REPORT)
}

// Report resources to goauth
func ReportResources(ec common.ExecContext) error {
	for _, v := range resources {
		if e := goauth.AddResource(ec.Ctx, v); e != nil {
			return fmt.Errorf("failed to call goauth.AddResource, %v", e)
		}
	}
	return nil
}
