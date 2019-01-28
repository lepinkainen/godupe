package main

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

func walkFunc(path string, info os.FileInfo, err error) error {
	// We can't do anything to directories
	if info.IsDir() {
		return nil
	}

	// dont re-hash existing files
	if Exists(path) {
		fmt.Printf("skipping: %s\n", path)
		return nil
	}

	fmt.Printf("hashing: %s\n", path)
	filename, hash := Hash(path)

	SaveHash(filename, hash)

	return nil
}

func main() {
	InitDB()
	//pruneDB()

	viper.AutomaticEnv()
	viper.SetDefault("GODUPE_DB", "./dupes.db")

	filepath.Walk(os.Args[1], walkFunc)
}
