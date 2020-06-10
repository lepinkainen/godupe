package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lepinkainen/godupe/db"
	"github.com/lepinkainen/godupe/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

func walkFunc(path string, info os.FileInfo, err error) error {
	// handle situations when a file isn't really a file or directory
	// usually files with really weird filenames on network drives
	defer func() {
		if x := recover(); x != nil {
			fmt.Printf("Unreadable file: %s\n", path)
			fmt.Println("Recovered in ", x)
		}
	}()

	// We can't do anything to directories
	if info.IsDir() {
		return nil
	}

	// dont re-hash existing files
	if db.Exists(path) {
		fmt.Printf("skipping: %s\n", path)
		return nil
	}

	// TODO: Add a goroutine for hashing in parallel?
	// TODO: Maybe with a configurable amount of workers and a limited channel size
	fmt.Printf("hashing: %s\n", path)
	filename, size, hash := file.Hash(path)

	if db.Dupe(hash) {
		fmt.Println("DUPE FOUND")
	}

	db.Save(filename, size, hash)

	return nil
}

func main() {
	db.Init()
	// TODO: only run if option provided
	// This WILL delete everything if a mount isn't available for example
	//Prune()

	viper.AutomaticEnv()
	viper.SetDefault("GODUPE_DB", "./dupes.db")

	// TODO: use cobra as a base for this
	fmt.Printf("Using database %s\n", viper.GetString("GODUPE_DB"))
	if len(os.Args) <= 1 {
		fmt.Println("usage: godupe [path]")
		return
	}
	filepath.Walk(os.Args[1], walkFunc)
}
