package file

import (
	"bufio"
	"os"

	log "github.com/sirupsen/logrus"
)

// Helper functions for cache management

func LoadCachedDirs() []string {
	var cachedDirs []string
	cacheFile, err := os.Open("cache.txt")
	if err == nil {
		defer cacheFile.Close()
		scanner := bufio.NewScanner(cacheFile)
		for scanner.Scan() {
			cachedDirs = append(cachedDirs, scanner.Text())
		}
	}
	return cachedDirs
}

func ContainsDir(dirs []string, dir string) bool {
	for _, cachedDir := range dirs {
		if cachedDir == dir {
			return true
		}
	}
	return false
}

func AddDirToCache(dir string) {
	cacheFile, err := os.OpenFile("cache.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer cacheFile.Close()
		_, err = cacheFile.WriteString(dir + "\n")
		if err != nil {
			log.Errorf("Error writing to cache: %s\n", err)
		}
	}
}
