package file

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	pb "github.com/cheggaaa/pb/v3"
)

// Hash a file, return its absolute path and SHA256
func Hash(filename string) (string, int64, string) {

	absfile, _ := filepath.Abs(filename)

	f, err := os.Open(absfile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, ""
	}

	// Create progress bar reader
	//bar := pb.New((int(info.Size())))
	bar := pb.Full.Start64(info.Size())
	bar.Set(pb.SIBytesPrefix, true)
	reader := bar.NewProxyReader(f)

	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		log.Fatal(err)
	}

	bar.Finish()

	return absfile, info.Size(), fmt.Sprintf("%x", h.Sum(nil))
}

// Exists returns true if the given file exists
func Exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}
