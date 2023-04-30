package fstore

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/curtisnewbie/goauth/client/goauth-client-go/gclient"
	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/gocommon/task"
	"github.com/gin-gonic/gin"
)

func init() {
	common.SetDefProp(PROP_ENABLE_GOAUTH_REPORT, false)
}

var (
	paths     = []gclient.CreatePathReq{}  // hardcoded paths for goauth
	resources = []gclient.AddResourceReq{} // hardcoded resources for goauth
)

const (
	CODE_FSTORE_UPLOAD = "fstore-upload"
)

func init() {
	common.SetDefProp(PROP_ENABLE_GOAUTH_REPORT, false)

	paths = append(paths, gclient.CreatePathReq{
		Type:   gclient.PT_PUBLIC,
		Url:    "/fstore/file/raw",
		Group:  "fstore",
		Desc:   "Fstore Raw File Download",
		Method: "GET",
	})
	paths = append(paths, gclient.CreatePathReq{
		Type:    gclient.PT_PROTECTED,
		Url:     "/fstore/file",
		Group:   "fstore",
		Desc:    "Fstore File Upload",
		Method:  "POST",
		ResCode: CODE_FSTORE_UPLOAD,
	})

	resources = append(resources, gclient.AddResourceReq{
		Name: "Fstore File Upload",
		Code: CODE_FSTORE_UPLOAD,
	})
}

func PrepareTasks() {
	task.ScheduleNamedDistributedTask("0 0 0/1 * * *", "PhyDelFileTask", func(ec common.ExecContext) {
		BatchPhyDelFiles(ec)
	})
}

func PrepareWebServer() {
	// TODO: supports file streaming (byte-range requests)

	// download file
	server.RawGet("/file/raw", func(c *gin.Context, ec common.ExecContext) {
		key := strings.TrimSpace(c.Query("key"))
		if key == "" {
			c.AbortWithStatus(404)
			return
		}

		if e := DownloadFileKey(ec, c.Writer, key); e != nil {
			ec.Log.Warnf("Failed to download by fileKey, %v", e)
			c.AbortWithStatus(404)
		}
	})

	// upload file
	server.Post("/file", func(c *gin.Context, ec common.ExecContext) (any, error) {
		fname := strings.TrimSpace(c.GetHeader("filename"))
		if fname == "" {
			return nil, common.NewWebErrCode(INVALID_REQUEST, "filename is required")
		}

		return UploadFile(ec, c.Request.Body, fname)
	})

	// get file's info
	server.Get("/file/info", func(c *gin.Context, ec common.ExecContext) (any, error) {
		fileId := strings.TrimSpace(c.Query("fileId"))
		if fileId == "" {
			return nil, common.NewWebErrCode(FILE_NOT_FOUND, FILE_NOT_FOUND)
		}
		f, ef := FindFile(fileId)
		if ef != nil {
			return nil, ef
		}
		if f.IsZero() {
			return f, common.NewWebErrCode(FILE_NOT_FOUND, "File is not found")
		}
		return f, nil
	})

	// generate random file key for the file
	server.Get("/file/key/random", func(c *gin.Context, ec common.ExecContext) (any, error) {
		fileId := strings.TrimSpace(c.Query("fileId"))
		if fileId == "" {
			return nil, common.NewWebErrCode(FILE_NOT_FOUND, "File is not found")
		}
		k, re := RandFileKey(ec, fileId)
		if re == nil {
			k = url.QueryEscape(k)
		}
		return k, re
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
		reportToGoAuth := func() {
			ec := common.EmptyExecContext()
			if e := ReportResources(ec); e != nil {
				ec.Log.Errorf("Failed to report resources, %v", e)
				return
			}
			if e := ReportPaths(ec); e != nil {
				ec.Log.Errorf("Failed to report paths, %v", e)
				return
			}
		}
		server.OnServerBootstrapped(reportToGoAuth)
	}
}

// Report paths to goauth
func ReportPaths(ec common.ExecContext) error {
	for _, v := range paths {
		if e := gclient.AddPath(ec.Ctx, v); e != nil {
			return fmt.Errorf("failed to call gclient.AddPath, %v", e)
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
		if e := gclient.AddResource(ec.Ctx, v); e != nil {
			return fmt.Errorf("failed to call gclient.AddResource, %v", e)
		}
	}
	return nil
}
