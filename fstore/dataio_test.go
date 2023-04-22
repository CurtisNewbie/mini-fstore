package fstore

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestCopyChkSum(t *testing.T) {
	inf := "test_TestCopy_in.txt"
	rf, ec := os.Create(inf)
	if ec != nil {
		t.Fatalf("Failed to create test file, %v", ec)
	}
	defer rf.Close()
	defer os.Remove(inf)

	ctn := "some stuff"
	_, ews := rf.WriteString(ctn)
	if ews != nil {
		t.Fatalf("Failed to write string to test file, %v", ews)
	}
	rf.Seek(0, io.SeekStart)

	outf := "test_TestCopy_out.txt"
	wf, ef := os.Create(outf)
	if ef != nil {
		t.Fatalf("Failed to create test file, %v", ef)
	}
	defer wf.Close()
	defer os.Remove(outf)

	n, md5, cce := CopyChkSum(rf, wf)
	if cce != nil {
		t.Fatalf("Failed to CopyChkSum, %v", cce)
	}

	if n < 1 {
		t.Fatalf("CopyChkSum return size < 1, %v", n)
	}
	expByteCnt := int64(len([]byte(ctn)))
	if n != expByteCnt {
		t.Fatalf("CopyChkSum return incorrect size, expected: %v, actual: %v", expByteCnt, n)
	}

	if strings.TrimSpace(md5) == "" {
		t.Fatalf("CopyChkSum return empty md5, %v", md5)
	}

	expMd5 := "beb6a43adfb950ec6f82ceed19beee21"
	if md5 != expMd5 {
		t.Fatalf("CopyChkSum return incorrect md5, expected: %v, actual: %v", expMd5, md5)
	}
}
