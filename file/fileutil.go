package file

import (
	"crypto/sha256"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"os"
	"path/filepath"

	pb "github.com/cheggaaa/pb/v3"
	"github.com/spf13/viper"
)

// Hash a file, return its absolute path, size and SHA256
func Hash(filename string) (string, int64, string) {
	partial := viper.GetBool("GODUPE_PARTIAL")
	partialSize := viper.GetInt64("GODUPE_PARTIAL_LIMIT")

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

	var hashSize int64
	if info.Size() < partialSize {
		hashSize = info.Size()
	} else {
		hashSize = partialSize
	}

	// Only use progress bar for files over this size
	//var useBar = info.Size() > (1 * 1000 * 1000)
	//useBar := true

	// Create progress bar reader
	var bar *pb.ProgressBar
	bar = pb.Simple.Start64(hashSize)
	bar.Set(pb.SIBytesPrefix, true)
	reader := bar.NewProxyReader(f)

	h := sha256.New()

	// Only do a partial hash
	if partial {
		if _, err := io.CopyN(h, reader, hashSize); err != nil {
			log.Fatal(err)
		}
	} else {
		if _, err := io.Copy(h, reader); err != nil {
			log.Fatal(err)
		}
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
