package fstore

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/redis"
	"github.com/curtisnewbie/gocommon/server"
)

func preTest(t *testing.T) {
	c := common.EmptyExecContext()
	ag := []string{"configFile=../app-conf-dev.yml"}
	common.DefaultReadConfig(ag, c)
	server.ConfigureLogging(c)
	common.SetProp(PROP_STORAGE_DIR, "../storage_test")
	common.SetProp(PROP_TRASH_DIR, "../trash_test")
	if err := mysql.InitMySqlFromProp(); err != nil {
		t.Fatal(err)
	}

	if _, err := redis.InitRedisFromProp(); err != nil {
		t.Fatal(err)
	}
}

func TestGenStoragePath(t *testing.T) {
	common.SetProp(PROP_STORAGE_DIR, "../storage_test")
	c := common.EmptyExecContext()
	p, eg := GenStoragePath(c, "file_123123")
	if eg != nil {
		t.Fatal(eg)
	}
	if p != "../storage_test/file_123123" {
		t.Fatalf("Generated path is incorrect, %s", p)
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

type PDelFileNoOp struct {
}

func (p PDelFileNoOp) delete(c common.ExecContext, fileId string) error {
	c.Log.Infof("Mock file delete, fileId: %v", fileId)
	return nil // do nothing
}

func TestListPendingPhyDelFiles(t *testing.T) {
	preTest(t)

	n := time.Now()
	c := common.EmptyExecContext()
	s, e := listPendingPhyDelFiles(c, n)
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("Listed: %v", s)
}

func TestBatchPhyDelFiles(t *testing.T) {
	preTest(t)
	c := common.EmptyExecContext()
	if e := BatchPhyDelFiles(c); e != nil {
		t.Fatal(e)
	}
}

func TestNewPDelFileOp(t *testing.T) {
	s := ""
	op := NewPDelFileOp(s)
	if op == nil {
		t.Fatal("op == nil")
	}
	if _, ok := op.(PDelFileTrashOp); !ok {
		t.Fatal("op should be PDelFileTrashOp")
	}

	s = "tttt"
	op = NewPDelFileOp(s)
	if op == nil {
		t.Fatal("op == nil")
	}
	if _, ok := op.(PDelFileTrashOp); !ok {
		t.Fatal("op should be PDelFileTrashOp")
	}

	s = "TRASH"
	op = NewPDelFileOp(s)
	if op == nil {
		t.Fatal("op == nil")
	}
	if _, ok := op.(PDelFileTrashOp); !ok {
		t.Fatal("op should be PDelFileTrashOp")
	}

	s = "direct"
	op = NewPDelFileOp(s)
	if op == nil {
		t.Fatal("op == nil")
	}
	if _, ok := op.(PDelFileDirectOp); !ok {
		t.Fatal("op should be PDelFileDirectOp")
	}

	s = "DIRECT"
	op = NewPDelFileOp(s)
	if op == nil {
		t.Fatal("op == nil")
	}
	if _, ok := op.(PDelFileDirectOp); !ok {
		t.Fatal("op should be PDelFileDirectOp")
	}
}

func TestPDelFileDirectOpt(t *testing.T) {
	common.SetProp(PROP_STORAGE_DIR, "../storage_test")
	common.SetProp(PROP_TRASH_DIR, "../trash_test")
	c := common.EmptyExecContext()

	fileId := "file_9876543210"
	fpath, eg := GenStoragePath(c, fileId)
	if eg != nil {
		t.Fatal(eg)
	}

	rf, ecr := os.Create(fpath)
	if ecr != nil {
		t.Fatalf("Failed to create test file, %v", ecr)
	}
	rf.Close()

	op := PDelFileDirectOp{}
	if ed := op.delete(c, fileId); ed != nil {
		t.Fatal(ed)
	}

	_, es := os.Stat(fpath)
	if es == nil {
		t.Fatal("File is not deleted")
	}

	if es != nil && !os.IsNotExist(es) {
		t.Fatalf("File cannot be deleted")
	}
}

func TestPDelFileTrashOpt(t *testing.T) {
	common.SetProp(PROP_STORAGE_DIR, "../storage_test")
	common.SetProp(PROP_TRASH_DIR, "../trash_test")
	c := common.EmptyExecContext()

	fileId := "file_9876543210"
	from, eg := GenStoragePath(c, fileId)
	if eg != nil {
		t.Fatal(eg)
	}

	rf, ecr := os.Create(from)
	if ecr != nil {
		t.Fatalf("Failed to create test file, %v", ecr)
	}
	rf.Close()

	op := PDelFileTrashOp{}
	if ed := op.delete(c, fileId); ed != nil {
		t.Fatal(ed)
	}

	to, eg := GenTrashPath(c, fileId)
	if eg != nil {
		t.Fatal(eg)
	}

	_, es := os.Stat(to)
	if es != nil {
		t.Fatalf("File not found, %v, %v", to, es)
	}
	os.Remove(to)
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

	err = PhyDelFile(ec, fileId, PDelFileNoOp{})
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

	testContent := "some stuff"

	fileId, eu := UploadFile(ec, bytes.NewReader([]byte(testContent)), "test.txt")
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

	p, _ := GenStoragePath(ec, fileId)
	os.Remove(p)
}

/*
func TestTransferFile(t *testing.T) {
	preTest(t)
	ec := common.EmptyExecContext()

	testContent := "some stuff"

	fileId, eu := UploadFile(ec, bytes.NewReader([]byte(testContent)), "test.txt")
	if eu != nil {
		t.Fatalf("Failed to upload file, %v", eu)
	}
	if fileId == "" {
		t.Fatalf("fileId is empty")
	}
	t.Logf("FileId: %v", fileId)

	p, _ := GenStoragePath(ec, fileId)
	defer os.Remove(p)

	fi, ef := FindFile(fileId)
	if ef != nil {
		t.Fatal(ef)
	}
	outbuf := bytes.NewBuffer([]byte{})

	et := TransferFile(ec, outbuf, fi, ZeroByteRange())
	if et != nil {
		t.Fatalf("Failed to transfer file, %v", et)
	}

	b, er := io.ReadAll(outbuf)
	if er != nil {
		t.Fatalf("Failed to read from output buffer, %v", er)
	}

	bs := string(b)
	if bs != testContent {
		t.Fatalf("Transferred file content mismatch, expected: %v, actual: %v", testContent, bs)
	}
	outbuf.Reset()

	et = TransferFile(ec, outbuf, fi, ByteRange{Start: 0, End: 2})
	if et != nil {
		t.Fatalf("Failed to transfer file, %v", et)
	}

	b, er = io.ReadAll(outbuf)
	if er != nil {
		t.Fatalf("Failed to read from output buffer, %v", er)
	}

	bs = string(b)
	expected := string([]byte(testContent)[0:3]) // 0-2 (inclusive)
	if bs != expected {
		t.Fatalf("Transferred file content mismatch, expected: %v, actual: %v", expected, bs)
	}
	t.Logf("Expected: %v", expected)
}
*/

func TestRandFileKey(t *testing.T) {
	preTest(t)
	ec := common.EmptyExecContext()
	k, er := RandFileKey(ec, "", "file_687330432057344050696")
	if er != nil {
		t.Fatal(er)
	}
	if k == "" {
		t.Fatalf("Generated fileKey is empty")
	}
}

func TestResolveFileId(t *testing.T) {
	preTest(t)
	fileId := "file_687330432057344050696"
	ec := common.EmptyExecContext()
	pname := "myfile.txt"
	k, er := RandFileKey(ec, pname, fileId)
	if er != nil {
		t.Fatal(er)
	}

	ok, resolved := ResolveFileKey(ec, k)
	if !ok {
		t.Fatal("Failed to resolve fileId")
	}
	if resolved.FileId != fileId {
		t.Fatalf("Resolved fileId doesn't match, expected: %s, actual: %s", fileId, resolved.FileId)
	}
	if resolved.Name != pname {
		t.Fatalf("Resolved name doesn't match, expected: %s, actual: %s", pname, resolved.Name)
	}
}

func TestAdjustByteRange(t *testing.T) {
	var br ByteRange
	var ea error

	// start == end
	br, ea = adjustByteRange(ByteRange{Start: 0, End: 0}, 100)
	if ea != nil {
		t.Fatal(ea)
	}
	if br.Start != 0 {
		t.Fatal("Start != 0")
	}
	if br.End != 0 {
		t.Fatal("End != 0")
	}
	if br.Size() != 1 {
		t.Fatal("Size != 1")
	}

	// start > end
	br, ea = adjustByteRange(ByteRange{Start: 1, End: 0}, 100)
	if ea == nil {
		t.Fatal("ea == nil")
	}
	t.Logf("ea: %v", ea)

	// start < end
	br, ea = adjustByteRange(ByteRange{Start: 0, End: 1}, 100)
	if ea != nil {
		t.Fatal(ea)
	}
	if br.Start != 0 {
		t.Fatal("Start != 0")
	}
	if br.End != 1 {
		t.Fatal("End != 1")
	}
	if br.Size() != 2 {
		t.Fatal("Size != 2")
	}

	// end > fileSize
	br, ea = adjustByteRange(ByteRange{Start: 0, End: 101}, 100)
	if ea != nil {
		t.Fatal(ea)
	}
	if br.Start != 0 {
		t.Fatal("Start != 0")
	}
	if br.End != 99 {
		t.Fatal("End != 99")
	}
	if br.Size() != 100 {
		t.Fatal("Size != 100")
	}

	// start < end, size() > BYTE_RANGE_MAX_SIZE
	br, ea = adjustByteRange(ByteRange{Start: 0, End: 40_000_000}, 50_000_000)
	if ea != nil {
		t.Fatal(ea)
	}
	if br.Size() != 30_000_000 {
		t.Fatalf("Size != 30_000_000, %v", br.Size())
	}
	if br.Start != 0 {
		t.Fatal("Start != 0")
	}
	if br.End != 29_999_999 {
		t.Fatalf("End != 29_999_999, %v", br.End)
	}

	// start < end, size() > BYTE_RANGE_MAX_SIZE
	br, ea = adjustByteRange(ByteRange{Start: 20_000_000, End: 55_000_000}, 60_000_000)
	if ea != nil {
		t.Fatal(ea)
	}
	if br.Size() != 30_000_000 {
		t.Fatalf("Size != 30_000_000, %v", br.Size())
	}
	if br.Start != 20_000_000 {
		t.Fatal("Start != 20_000_000")
	}
	if br.End != 49_999_999 {
		t.Fatalf("End != 49_999_999, %v", br.End)
	}

}

func TestSanitizeStorage(t *testing.T) {
	preTest(t)
	ec := common.EmptyExecContext()
	common.SetProp(PROP_SANITIZE_STORAGE_TASK_DRY_RUN, true)
	common.SetProp(PROP_STORAGE_DIR, "../storage")
	common.SetProp(PROP_TRASH_DIR, "../trash")
	e := SanitizeStorage(ec)
	if e != nil {
		t.Fatal(e)
	}
}
