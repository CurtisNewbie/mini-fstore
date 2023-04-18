package fstore

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate random string
func Rand(len int) (string, error) {
	if len < 1 {
		return "", nil
	}

	// each base64 character represent 6 bits of data
	c := len*3/4 // wihtout padding
	b := make([]byte, c)
	_, e := rand.Read(b)
	if e != nil {
		return "", e
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}
