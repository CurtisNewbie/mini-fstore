package client

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/miso/miso"
)

var (
	ErrFileNotFound  = errors.New("file not found")
	ErrFileDeleted   = errors.New("file deleted")
	ErrIllegalFormat = errors.New("illegal format")

	ErrMapper = map[string]error{
		api.FileNotFound:  ErrFileNotFound,
		api.FileDeleted:   ErrFileDeleted,
		api.IllegalFormat: ErrIllegalFormat,
	}
)

func FetchFileInfo(rail miso.Rail, req api.FetchFileInfoReq) (api.FstoreFile, error) {
	var r miso.GnResp[api.FstoreFile]
	err := miso.NewDynTClient(rail, "/file/info", "fstore").
		Require2xx().
		AddQueryParams("fileId", req.FileId).
		AddQueryParams("uploadFileId", req.UploadFileId).
		Get().
		Json(&r)

	if err != nil {
		return api.FstoreFile{}, fmt.Errorf("failed to fetch mini-fstore fileInfo, %w", err)
	}
	return r.MappedRes(ErrMapper)
}

func DeleteFile(rail miso.Rail, fileId string) error {
	var r miso.GnResp[any]
	err := miso.NewDynTClient(rail, "/file", "fstore").
		Require2xx().
		AddQueryParams("fileId", fileId).
		Delete().
		Json(&r)
	if err != nil {
		return fmt.Errorf("failed to delete mini-fstore file, fileId: %v, %v", fileId, err)
	}

	_, err = r.MappedRes(ErrMapper)
	return err
}

func GenTempFileKey(rail miso.Rail, fileId string, filename string) (string, error) {
	var r miso.GnResp[string]
	err := miso.NewDynTClient(rail, "/file/key", "fstore").
		Require2xx().
		AddQueryParams("fileId", fileId).
		AddQueryParams("filename", url.QueryEscape(filename)).
		Get().
		Json(&r)
	if err != nil {
		return "", fmt.Errorf("failed to generate mini-fstore temp token, fileId: %v, filename: %v, %v",
			fileId, filename, err)
	}

	return r.MappedRes(ErrMapper)
}

func DownloadFile(rail miso.Rail, tmpToken string, writer io.Writer) error {
	_, err := miso.NewDynTClient(rail, "/file/raw", "fstore").
		EnableTracing().
		AddQueryParams("key", tmpToken).
		Get().
		WriteTo(writer)
	return err
}

func UploadFile(rail miso.Rail, filename string, dat io.Reader) (string /* uploadFileId */, error) {
	var res miso.GnResp[string]
	err := miso.NewDynTClient(rail, "/file", "fstore").
		EnableTracing().
		AddHeaders(map[string]string{"filename": filename}).
		Put(dat).
		Json(&res)
	if err != nil {
		return "", fmt.Errorf("failed to UploadFstoreFile, filename: %v, %v", filename, err)
	}
	return res.Res()
}

func TriggerFileUnzip(rail miso.Rail, req api.UnzipFileReq) error {
	var r miso.GnResp[any]
	err := miso.NewDynTClient(rail, "/file/unzip", "fstore").
		Require2xx().
		PostJson(req).
		Json(&r)
	if err != nil {
		return fmt.Errorf("failed to trigger mini-fstore unzip pipeline, req: %+v, %v", req, err)
	}
	_, err = r.MappedRes(ErrMapper)
	return err
}
