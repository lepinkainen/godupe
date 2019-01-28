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
	// We can't do anything to directories
	if info.IsDir() {
		return nil
	}

	// dont re-hash existing files
	if db.Exists(path) {
		fmt.Printf("skipping: %s\n", path)
		return nil
	}

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

	fmt.Printf("Using database %s", viper.GetString("GODUPE_DB"))
	filepath.Walk(os.Args[1], walkFunc)
}
