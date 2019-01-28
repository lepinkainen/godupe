package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	pb "gopkg.in/cheggaaa/pb.v1"
)

// Hash a file, return its absolute path and SHA256
func Hash(filename string) (string, string) {

	absfile, _ := filepath.Abs(filename)

	f, err := os.Open(absfile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", ""
	}

	h := sha256.New()
	bar := pb.New((int(info.Size()))).SetUnits(pb.U_BYTES)
	bar.Start()
	reader := bar.NewProxyReader(f)

	if _, err := io.Copy(h, reader); err != nil {
		log.Fatal(err)
	}

	bar.Finish()

	return absfile, fmt.Sprintf("%x", h.Sum(nil))
}

// FileExists returns true if the given file exists
func FileExists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}
