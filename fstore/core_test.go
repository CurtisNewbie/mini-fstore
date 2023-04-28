package fstore

import (
	"io"
	"os"
	"testing"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/redis"
)

func preTest(t *testing.T) {
	ag := []string{"configFile=../app-conf-dev.yml"}
	common.DefaultReadConfig(ag)
	common.SetProp(PROP_STORAGE_DIR, "../storage_test")
	if err := mysql.InitMySqlFromProp(); err != nil {
		t.Fatal(err)
	}

	if _, err := redis.InitRedisFromProp(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateFileRec(t *testing.T) {
	preTest(t)

	ec := common.EmptyExecContext()
	fileId := GenFileId()

	err := CreateFileRec(ec, CreateFile{
		FileId: fileId,
		Name:   "test.txt",
		Size:   10,
		Md5:    "HAKLJGHSLKDFJS",
	})
	if err != nil {
		t.Fatalf("Failed to create file record, %v", err)
	}
	t.Logf("FileId: %v", fileId)
}

func TestLDelFile(t *testing.T) {
	preTest(t)

	ec := common.EmptyExecContext()
	fileId := GenFileId()

	err := CreateFileRec(ec, CreateFile{
		FileId: fileId,
		Name:   "test.txt",
		Size:   10,
		Md5:    "HAKLJGHSLKDFJS",
	})
	if err != nil {
		t.Fatalf("Failed to create file record, %v", err)
	}

	err = LDelFile(ec, fileId)
	if err != nil {
		t.Fatalf("Failed to LDelFile, %v", err)
	}
}

func TestPhyDelFile(t *testing.T) {
	preTest(t)

	ec := common.EmptyExecContext()
	fileId := GenFileId()

	err := CreateFileRec(ec, CreateFile{
		FileId: fileId,
		Size:   10,
		Md5:    "HAKLJGHSLKDFJS",
	})
	if err != nil {
		t.Fatalf("Failed to create file record, %v", err)
	}

	err = PhyDelFile(ec, fileId)
	if err != nil {
		t.Fatalf("Failed PhyDelFile, %v", err)
	}
}

func TestListLDelFile(t *testing.T) {
	preTest(t)

	ec := common.EmptyExecContext()
	l, e := ListLDelFile(ec, 0, 1)
	if e != nil {
		t.Fatalf("failed to ListLDelFile, %v", e)
	}
	if len(l) < 1 {
		t.Fatalf("should have found at least one ldel file, actual: %d", len(l))
	}
	t.Logf("Found ldel file: %v", l)
}

func TestUploadFile(t *testing.T) {
	preTest(t)
	ec := common.EmptyExecContext()

	inf := "test_TestUploadFile_in.txt"
	rf, ecr := os.Create(inf)
	if ecr != nil {
		t.Fatalf("Failed to create test file, %v", ec)
	}
	defer rf.Close()
	defer os.Remove(inf)

	_, ews := rf.WriteString("some stuff")
	if ews != nil {
		t.Fatalf("Failed to write string to test file, %v", ews)
	}
	rf.Seek(0, io.SeekStart)

	fileId, eu := UploadFile(ec, rf, "test.txt")
	if eu != nil {
		t.Fatalf("Failed to upload file, %v", eu)
	}
	if fileId == "" {
		t.Fatalf("fileId is empty")
	}
	t.Logf("FileId: %v", fileId)

	f, ef := FindFile(fileId)
	if ef != nil {
		t.Fatalf("Failed to find file, %v", ef)
	}

	expMd5 := "beb6a43adfb950ec6f82ceed19beee21"
	if f.Md5 != expMd5 {
		t.Fatalf("UploadFile saved incorrect md5, expected: %v, actual: %v", expMd5, f.Md5)
	}

	p, _ := GenFilePath(fileId)
	os.Remove(p)
}

func TestDownloadFile(t *testing.T) {
	preTest(t)
	ec := common.EmptyExecContext()

	inf := "test_TestDownFile_in.txt"
	rf, ecr := os.Create(inf)
	if ecr != nil {
		t.Fatalf("Failed to create test file, %v", ec)
	}
	defer rf.Close()
	defer os.Remove(inf)

	testContent := "some stuff"
	_, ews := rf.WriteString(testContent)
	if ews != nil {
		t.Fatalf("Failed to write string to test file, %v", ews)
	}
	rf.Seek(0, io.SeekStart)

	fileId, eu := UploadFile(ec, rf, "test.txt")
	if eu != nil {
		t.Fatalf("Failed to upload file, %v", eu)
	}
	if fileId == "" {
		t.Fatalf("fileId is empty")
	}
	t.Logf("FileId: %v", fileId)

	p, _ := GenFilePath(fileId)
	defer os.Remove(p)

	outf := "test_TestDownFile_out.txt"
	of, eof := os.Create(outf)
	if eof != nil {
		t.Fatalf("Failed to create test file, %v", eof)
	}
	defer os.Remove(outf)

	ed := DownloadFile(ec, of, fileId)
	if ed != nil {
		t.Fatalf("Failed to download file, %v", ed)
	}

	of.Seek(0, io.SeekStart)
	b, er := io.ReadAll(of)
	if er != nil {
		t.Fatalf("Failed to read output file, %v", er)
	}

	bs := string(b)
	if bs != testContent {
		t.Fatalf("Downloaded file content mismatch, expected: %v, actual: %v", testContent, bs)
	}
}

func TestRandFileKey(t *testing.T) {
	preTest(t)
	ec := common.EmptyExecContext()
	k, er := RandFileKey(ec, "file_676106983194624208429")
	if er != nil {
		t.Fatal(er)
	}
	if k == "" {
		t.Fatalf("Generated fileKey is empty")
	}
}

func TestResolveFileId(t *testing.T) {
	preTest(t)
	fileId := "file_676106983194624208429"
	ec := common.EmptyExecContext()
	k, er := RandFileKey(ec, fileId)
	if er != nil {
		t.Fatal(er)
	}

	ok, resolved := ResolveFileId(ec, k)
	if !ok {
		t.Fatal("Failed to resolve fileId")
	}
	if resolved != fileId {
		t.Fatalf("Resolved fileId doesn't match, expected: %s, actual: %s", fileId, resolved)
	}
}
