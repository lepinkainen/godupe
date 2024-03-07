package file

import (
	"crypto/sha256"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
)

// Helper function to list files in a directory and get their absolute paths
func WalkDirFiles(dirPath string) ([]string, error) {
	files := []string{}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			absPath, err := filepath.Abs(filepath.Join(dirPath, entry.Name()))
			if err != nil {
				return nil, err
			}
			files = append(files, absPath)
		}
	}
	return files, nil
}

// Hash a file, return its absolute path, size and SHA256
func Hash(filename string) (string, int64, string, error) {
	partial := viper.GetBool("partial")
	partialSize := viper.GetInt64("limit") * 1048576

	absfile, _ := filepath.Abs(filename)

	f, err := os.Open(absfile)
	if err != nil {
		log.Error(err)
		return "", 0, "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, "", err
	}

	/*
		if info.Size() < 10*1024*1024 {
			log.Debugf("Skipping small file: %s\n", filename)
			// return error
			return "", 0, "", fmt.Errorf("skipping small file: %s", filename)
		}
	*/

	var hashSize int64
	// If file is smaller than partial size, don't try to read more than the file's size
	if info.Size() < partialSize {
		hashSize = info.Size()
	} else {
		hashSize = partialSize
	}

	// Only use progress bar for files over this size
	//var useBar = info.Size() > (1 * 1000 * 1000)
	//useBar := true

	// Create progress bar reader
	bar := progressbar.DefaultBytes(info.Size())
	bar.Describe(filename)

	h := sha256.New()

	// Only do a partial hash
	if partial {
		if _, err := io.CopyN(io.MultiWriter(h, bar), f, hashSize); err != nil {
			log.Fatal(err)
		}
	} else {
		if _, err := io.Copy(io.MultiWriter(h, bar), f); err != nil {
			log.Fatal(err)
		}
	}

	bar.Finish()

	return absfile, info.Size(), fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Exists returns true if the given file exists
func Exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

// Helper function to check for subdirectories
func HasSubdirectories(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	names, err := f.Readdirnames(0) // Get a list of directory entries
	if err != nil {
		return false, err
	}

	for _, name := range names {
		finfo, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			return false, err
		}
		if finfo.IsDir() {
			return true, nil // Found a subdirectory
		}
	}

	return false, nil // No subdirectories found
}
