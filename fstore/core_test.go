package fstore

import (
	"testing"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/redis"
)

func preTest(t *testing.T) {
	ag := []string{"configFile=../app-conf-dev.yml"}
	common.DefaultReadConfig(ag)
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
