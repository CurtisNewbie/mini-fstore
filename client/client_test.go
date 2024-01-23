package client

import (
	"errors"
	"os"
	"testing"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/miso/miso"
)

func _clientPreTest(t *testing.T) miso.Rail {
	miso.SetProp(miso.PropAppName, "test")
	miso.SetProp("client.addr.fstore.host", "localhost")
	miso.SetProp("client.addr.fstore.port", "8084")
	return miso.EmptyRail()
}

func TestFetchFileInfo(t *testing.T) {
	rail := _clientPreTest(t)
	ff, err := FetchFileInfo(rail, api.FetchFileInfoReq{
		FileId: "file_1049792900153344189170",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", ff)

	_, err = FetchFileInfo(rail, api.FetchFileInfoReq{
		FileId: "123",
	})
	if err == nil {
		t.Fatal("should return error")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatal("should return ErrFileNotFound")
	}
}

func TestDeleteFile(t *testing.T) {
	rail := _clientPreTest(t)
	err := DeleteFile(rail, "file_1049793827766272189170")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenTempFileKey(t *testing.T) {
	rail := _clientPreTest(t)
	key, err := GenTempFileKey(rail, "file_1052177215258624996902", "123.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if key == "" {
		t.Fatal("should generate key")
	}
	t.Log(key)
}

func TestUploadFile(t *testing.T) {
	rail := _clientPreTest(t)
	f, err := miso.OpenFile("../conf.yml", os.O_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fileId, err := UploadFile(rail, "conf.yml", f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fileId)
}

func TestDownloadFile(t *testing.T) {
	rail := _clientPreTest(t)
	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tempKey := "WRgueucOHfvwxzdEgAlxBy+6ro6IVw"
	err = DownloadFile(rail, tempKey, f)
	if err != nil {
		t.Fatal(err)
	}
	buf, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(buf))
}

func TestTriggerUnzipFilePipeline(t *testing.T) {
	rail := _clientPreTest(t)
	miso.SetLogLevel("debug")

	err := TriggerFileUnzip(rail, api.UnzipFileReq{
		FileId:          "file_1062109045440512875450",
		ReplyToEventBus: "testunzip",
	})
	if err != nil {
		t.Fatal(err)
	}
}
