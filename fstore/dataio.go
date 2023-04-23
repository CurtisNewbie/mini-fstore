package fstore

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	DEFAULT_BUFFER_SIZE = int64(64 * 1024)
)

// Create buffer with default size
func DefBuf() []byte {
	bufSize := DEFAULT_BUFFER_SIZE
	return make([]byte, bufSize)
}

// CopyChkSum copy data from reader to writer and calculate the md5 checksum on the fly.
//
// return the transferred size in bytes and the md5 checksum
func CopyChkSum(r io.Reader, w io.Writer) (int64, string, error) {
	buf := DefBuf()
	size := int64(0)
	hash := md5.New()

	for {
		nr, er := r.Read(buf)

		if er != nil {
			if er != io.EOF {
				return size, "", fmt.Errorf("failed to read from Reader, %v", er)
			}
			break // EOF
		}

		if nr > 0 {
			nw, ew := w.Write(buf[0:nr])
			if ew != nil {
				return size, "", fmt.Errorf("failed to write to Writer, %v", ew)
			}
			if nw < 0 || nr != nw {
				return size, "", fmt.Errorf("invalid writer.Write returned values, expected write: %v, actual write: %v", nr, nw)
			}
			size += int64(nw)

			nh, eh := hash.Write(buf[0:nr])
			if eh != nil {
				return size, "", fmt.Errorf("failed to write to md5 hash writer, %v", eh)
			}
			if nh < 0 || nr != nh {
				return size, "", fmt.Errorf("invalid md5 hash writer.Write returned values, expected write: %v, actual write: %v", nr, nh)
			}

		}
	}
	return size, hex.EncodeToString(hash.Sum(nil)), nil
}
