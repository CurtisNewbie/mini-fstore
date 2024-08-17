package fstore

import (
	"crypto/md5"
	"fmt"
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

	n, checksum, cce := CopyChkSum(rf, wf)
	if len(checksum) < 2 {
		t.Fatalf("checksum.len < 2, %v", len(checksum))
	}
	t.Logf("checksum: %#v", checksum)
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

	md5 := checksum["md5"].Hex
	if strings.TrimSpace(md5) == "" {
		t.Fatalf("CopyChkSum return empty md5")
	}

	expMd5 := "beb6a43adfb950ec6f82ceed19beee21"
	if md5 != expMd5 {
		t.Fatalf("CopyChkSum return incorrect md5, expected: %v, actual: %v", expMd5, md5)
	}
}

func TestMultiCopyChkSum(t *testing.T) {
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

	cleanUpFiles := true
	outFiles := []io.Writer{}
	for i := 0; i < 10; i++ {
		outf := fmt.Sprintf("test_TestCopy_out_%d_.txt", i)
		wf, ef := os.Create(outf)
		if ef != nil {
			t.Fatalf("Failed to create test file, %v", ef)
		}
		outFiles = append(outFiles, wf)
		defer wf.Close()

		if cleanUpFiles {
			defer os.Remove(outf)
		}
	}

	hashing := []Hashing{{Name: "md5", Hash: md5.New()}}
	n, checksum, cce := MultiCopyChkSum(rf, hashing, outFiles...)
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

	md5 := checksum[0].Hex
	if strings.TrimSpace(md5) == "" {
		t.Fatalf("CopyChkSum return empty md5, %v", md5)
	}

	expMd5 := "beb6a43adfb950ec6f82ceed19beee21"
	if md5 != expMd5 {
		t.Fatalf("CopyChkSum return incorrect md5, expected: %v, actual: %v", expMd5, md5)
	}
}
