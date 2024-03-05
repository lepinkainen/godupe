/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lepinkainen/godupe/db"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check the given tree for existing files",
	Run:   check,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// checkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// checkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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

	checkCmd.Flags().String("db", dbPath, "DB file to use")
}

func checkWalkFunc(path string, info os.FileInfo, err error) error {
	// handle situations when a file isn't really a file or directory
	// usually files with really weird filenames on network drives
	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Unreadable file: %s\n", path)
			log.Errorf("Recovered in %s", x)
		}
	}()

	// We can't do anything to directories
	if info.IsDir() {
		return nil
	}

	// TODO: maybe load the full list of stuff to memory to speed up the process?
	// Benchmark it?
	absfilepath, _ := filepath.Abs(path)
	res := db.Exists(absfilepath)
	if res == db.HashTypeNotExist {
		fmt.Printf("Not found: %s\n", path)
		return nil
	}
	fmt.Printf("Found: %s\n", path)

	return nil
}

func check(cmd *cobra.Command, args []string) {
	viper.AutomaticEnv()

	viper.BindPFlag("db", cmd.Flags().Lookup("db"))

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	log.Infof("Using database %s\n", viper.GetString("db"))

	db.Init()
	filepath.Walk(args[0], checkWalkFunc)
}
