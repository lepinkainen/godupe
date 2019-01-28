package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// InitDB initializes the database
func InitDB() {
	db, err := sql.Open("sqlite3", viper.GetString("GODUPE_DB"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := "create table dupes (path text not null primary key, hash text, date);"

	_, err = db.Exec(sqlStmt)
	if err != nil {
		// Ignore if table is already created
		if err.Error() == "table dupes already exists" {
			return
		}
		log.Printf("%q: %s\n", err, sqlStmt)
	}
}

// PruneDB delete files that don't exist any more
func PruneDB() {
	db, err := sql.Open("sqlite3", viper.GetString("GODUPE_DB"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("select path from dupes")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var pruneList []string

	for rows.Next() {
		var filename string
		err = rows.Scan(&filename)
		if err != nil {
			log.Fatal(err)
		}
		if !FileExists(filename) {
			pruneList = append(pruneList, filename)
		}
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := db.Prepare("delete from dupes where path = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, filename := range pruneList {
		fmt.Printf("Pruned: %s\n", filename)
		stmt.Exec(filename)
	}
}

// Exists returns true if file has already been hashed
func Exists(filename string) bool {
	db, err := sql.Open("sqlite3", viper.GetString("GODUPE_DB"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("select count(*) from dupes where path = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var count int
	err = stmt.QueryRow(filename).Scan(&count)
	if err != nil {
		return false
	}

	if count > 0 {
		return true
	}

	return false
}

// SaveHash stores the file hash to the database
func Save(filename string, size int64, hash string) {
	db, err := sql.Open("sqlite3", viper.GetString("GODUPE_DB"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	// upsert path and hash
	stmt, err := tx.Prepare("insert into dupes(path, hash, date) values(?, ?, CURRENT_TIMESTAMP) on conflict(path) do update set hash=?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(filename, hash, hash)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}
