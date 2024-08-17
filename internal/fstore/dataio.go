package fstore

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
)

const (
	DEFAULT_BUFFER_SIZE = int64(64 * 1024)
)

// Create buffer with default size
func DefBuf() []byte {
	bufSize := DEFAULT_BUFFER_SIZE
	return make([]byte, bufSize)
}

type Hashing struct {
	Hash hash.Hash
	Name string
}

type Checksum struct {
	Name string
	Hex  string
}

// CopyChkSum copy data from reader to multiple writer(s) and calculate the hash checksum on the fly.
//
// return the transferred size in bytes and the hash checksum
func MultiCopyChkSum(r io.Reader, hashing []Hashing, ws ...io.Writer) (int64, []Checksum, error) {
	buf := DefBuf()
	size := int64(0)

	for {
		nr, er := r.Read(buf)

		if nr > 0 {
			for _, h := range hashing {
				// write to hash first
				nh, eh := h.Hash.Write(buf[0:nr])
				if eh != nil {
					return size, nil, fmt.Errorf("failed to write to %v hash writer, %v", h.Name, eh)
				}
				if nh < 0 || nr != nh {
					return size, nil, fmt.Errorf("invalid %v hash writer.Write returned values, expected write: %v, actual write: %v", h.Name, nr, nh)
				}
			}

			// update size
			size += int64(nr)

			// writer to all the writers one by one
			for iw, w := range ws {
				nw, ew := w.Write(buf[0:nr])
				if ew != nil {
					return size, nil, fmt.Errorf("failed to write to Writer[%d], %v", iw, ew)
				}
				if nw < 0 || nr != nw {
					return size, nil, fmt.Errorf("invalid writer.Write[%d] returned values, expected write: %v, actual write: %v", iw, nr, nw)
				}
			}
		}

		// it's possible that the r.Read() returns non zero nr and a non-nil er at the same time
		if er != nil {
			if er != io.EOF {
				return size, nil, fmt.Errorf("failed to read from Reader, %v", er)
			}
			break // EOF
		}
	}

	checksum := make([]Checksum, 0, len(hashing))
	for _, h := range hashing {
		checksum = append(checksum, Checksum{Name: h.Name, Hex: hex.EncodeToString(h.Hash.Sum(nil))})
	}

	return size, checksum, nil
}

// CopyChkSum copy data from reader to writer and calculate hash on the fly.
//
// return the transferred size in bytes and the md5 checksum
func CopyChkSum(r io.Reader, w io.Writer) (int64, map[string]Checksum, error) {
	hashing := []Hashing{{Name: "md5", Hash: md5.New()}, {Name: "sha1", Hash: sha1.New()}}
	n, cs, err := MultiCopyChkSum(r, hashing, w)
	if err != nil {
		return n, nil, err
	}
	m := make(map[string]Checksum, len(hashing))
	for i := range cs {
		v := cs[i]
		m[v.Name] = Checksum{Name: v.Name, Hex: v.Hex}
	}
	return n, m, err
}

func ChkSumSha1(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
