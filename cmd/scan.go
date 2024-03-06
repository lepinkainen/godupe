/*
Copyright © 2020 Riku Lindblad <riku.lindblad@iki.fi>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/lepinkainen/godupe/db"
	"github.com/lepinkainen/godupe/file"

	// We're using sqlite for the DB
	_ "github.com/mattn/go-sqlite3"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan [directory]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Scan and add file checksums to DB",
	Run:   scan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// scanCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Get user's configuration directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Error getting configuration directory:", err)
		return
	}

	// Create the godupe directory if it doesn't exist
	godupeDir := filepath.Join(configDir, "godupe")
	err = os.MkdirAll(godupeDir, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}

	// Construct the full path to the database file
	dbPath := filepath.Join(godupeDir, "godupe.db")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	scanCmd.Flags().BoolP("partial", "p", false, "Only read the first X MiB of a file to generate a partial hash")
	scanCmd.Flags().String("db", dbPath, "DB file to use")
	scanCmd.Flags().Int64("limit", 2, "Amount of MiB to read when doing partial scan")
}

func walkDirFunc(path string, d fs.DirEntry, err error) error {
	if err != nil {
		log.Errorf("Error accessing directory: %s\n", path)
		return err
	}

	// Skip if all files are already in database
	if d.IsDir() {
		files, err := file.WalkDirFiles(path)
		log.Infof("processing: %s [%d files]\n", path, len(files))
		if err != nil {
			return err
		}

		skip := db.ExistsAll(files)

		if skip {
			log.Debug("-> Skip - already processed")
			return filepath.SkipDir
		}

		return nil
	}

	// Handle potential panics during file access
	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Unreadable file: %s\n", path)
			log.Errorf("Recovered in %s", x)
		}
	}()

	absfilepath, err := filepath.Abs(path)
	if err != nil {
		log.Errorf("Error getting absolute path for %s: %s\n", path, err)
		return err
	}

	partial := viper.GetBool("partial")

	// Check if file already exists based on hash type
	res := db.Exists(absfilepath)
	if (partial && (res == db.HashTypePartial || res == db.HashTypeFull)) ||
		(!partial && res == db.HashTypeFull) {
		log.Debugf("skipping: %s\n", path)
		return nil
	}

	// Perform file operations
	filename, size, hash, err := file.Hash(path)
	/*
		if err != nil {
			log.Errorf("Error hashing file %s: %s\n", path, err)
			return err
		}
	*/

	// Skip empty files
	if size == 0 {
		log.Debugf("skipping empty file: %s\n", path)
		return nil
	}

	db.Save(filename, size, hash)

	return nil
}

func walkFunc(path string, info os.FileInfo, err error) error {
	// handle situations when a file isn't really a file or directory
	// usually files with really weird filenames on network drives
	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Unreadable file: %s\n", path)
			log.Errorf("Recovered in %s", x)
		}
	}()
	partial := viper.GetBool("partial")

	if err != nil {
		log.Errorf("Error walking directory: %s\n", path)
		return err
	}

	// We can't do anything to directories
	if info.IsDir() {
		log.Infof("processing: %s\n", path)
		return nil
	}

	// TODO: maybe load the full list of stuff to memory to speed up the process?
	// TODO: is is possible to remember stuff in a recursive function?
	// Benchmark it?
	absfilepath, _ := filepath.Abs(path)
	res := db.Exists(absfilepath)
	// if we are doing partial hashes and partial or full hash exists, skip

	if partial {
		if res == db.HashTypePartial || res == db.HashTypeFull {
			log.Debugf("skipping: %s\n", path)
			return nil
		}
	} else {
		// for full hash mode, we require the full hash, if we only have a partial, recalculate
		if res == db.HashTypeFull {
			log.Debugf("skipping: %s\n", path)
			return nil
		}
	}

	// TODO: Add a goroutine for hashing in parallel?
	// TODO: Maybe with a configurable amount of workers and a limited channel size
	if partial {
		log.Debugf("hashing (partial): %s\n", path)
	} else {
		log.Debugf("hashing: %s\n", path)

	}
	filename, size, hash, err := file.Hash(path)
	// TODO: This is a complete hack, the f.Stat call should be done here
	// skip small files for now
	if size == 0 {
		log.Debugf("skipping empty file: %s\n", path)
		return nil
	}

	db.Save(filename, size, hash)

	return nil
}

func scan(cmd *cobra.Command, args []string) {

	viper.AutomaticEnv()

	viper.BindPFlag("partial", cmd.Flags().Lookup("partial"))
	viper.BindPFlag("db", cmd.Flags().Lookup("db"))
	viper.BindPFlag("limit", cmd.Flags().Lookup("limit"))

	//const mib = 1048576 // 1 MiB
	//const partialSize = 2 * mib

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	log.Infof("Using database %s\n", viper.GetString("db"))
	if viper.GetBool("partial") {
		log.Infoln("Running partial scan")
	}

	db.Init()
	//filepath.Walk(args[0], walkFunc)
	filepath.WalkDir(args[0], walkDirFunc)
}
