package db

import (
	"database/sql"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Init initializes the database
func Init() {
	db, err := sql.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Debugf("Initializing DB in %s", viper.GetString("db"))

	// Create dupes table
	sqlStmt := "CREATE TABLE IF NOT EXISTS dupes (path text not null primary key, hash text, partialhash text, date);"

	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Errorf("%q: %s\n", err, sqlStmt)
	}

	// we're doing a ton of operations on the path column, index it to aid performance a bit
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_path ON dupes (path);")
	if err != nil {
		log.Errorf("%q: %s\n", err, sqlStmt)
	}

}

// Prune deletes files that don't exist any more
func Prune() {
	db, err := sql.Open("sqlite3", viper.GetString("db"))
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

	// TODO: Progress bar?
	for rows.Next() {
		var filename string
		err = rows.Scan(&filename)
		if err != nil {
			log.Fatal(err)
		}
		// File is in DB, but not in filesystem
		if Exists(filename) != HashTypeNotExist {
			fmt.Printf("Pruning %s\n", filename)
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

// Dupe returns true if file has already been hashed
func Dupe(hash, partialhash string) bool {
	db, err := sql.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("select count(*) from dupes where hash = ? or partialhash = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var count int
	err = stmt.QueryRow(hash, partialhash).Scan(&count)
	if err != nil {
		return false
	}

	if count > 0 {
		return true
	}

	return false
}

// HashType stores the way the file has been hashed
type HashType string

const (
	// HashTypeNotExist Hash not in DB
	HashTypeNotExist HashType = "NOTEXIST"
	// HashTypeNone = not hashed
	HashTypeNone HashType = "NONE"
	// HashTypeFull = full file hashed
	HashTypeFull HashType = "FULL"
	// HashTypePartial = First X MB of file hashed
	HashTypePartial HashType = "PARTIAL"
)

// Check the files in batches
// TODO: Dis no worky
func ExistsAllBatch(filenames []string) bool {

	const batchMax = 10

	db, err := sql.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for i := 0; i < len(filenames); i += batchMax {
		// Limit the batch size to 10
		batchSize := min(batchMax, len(filenames)-i)
		batch := filenames[i : i+batchSize]

		// Build the IN clause
		inClause := buildInClause(batch)

		stmt, err := db.Prepare("select path from dupes where path IN " + inClause)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		// Execute the query with the IN clause
		rows, err := stmt.Query(batch)

		if err != nil {
			log.Errorf("error checking files: %v", err)
			return false
		}
		defer rows.Close()

		// Count rows using `Next` loop
		count := 0
		for rows.Next() {
			count++
		}

		if count != batchSize {
			return false
		}
	}

	return true
}

// Helper function to build the IN clause string
func buildInClause(filenames []string) string {
	inClause := "("
	for i := range filenames {
		if i > 0 {
			inClause += ","
		}
		inClause += "?"
	}
	inClause += ")"
	return inClause
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ExistsAll(filenames []string) bool {
	db, err := sql.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("select path, hash, partialhash from dupes where path = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, filename := range filenames {
		var path, hash, partialhash string
		row := stmt.QueryRow(filename)
		err := row.Scan(&path, &hash, &partialhash)
		if err == sql.ErrNoRows {
			// File not found in database
			return false
		}
		// In database, but no hash -> we need to calculate it
		if hash == "" {
			return false
		}
		// Only check for existence, ignore hash types
	}

	// All files found
	return true
}

// Exists returns true if file has already been hashed
func Exists(filename string) HashType {
	db, err := sql.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("select path, hash, partialhash from dupes where path = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var path, hash, partialhash string
	row := stmt.QueryRow(filename)
	err = row.Scan(&path, &hash, &partialhash)
	if err == sql.ErrNoRows {
		// No row returned, not hashed
		return HashTypeNotExist
	}

	// Full hash, no need for partial
	if hash != "" {
		return HashTypeFull
	}
	if partialhash != "" {
		return HashTypePartial
	}

	// In DB but not hashed
	return HashTypeNone
}

// Save stores the file and its metadata to the DB
func Save(filename string, size int64, hash string) {
	partial := viper.GetBool("partial")
	partialSize := viper.GetInt64("limit") * 1048576

	// TODO: In partial mode if size < partial limit, save partial hash also in full hash
	// Maybe recurse the func and do two saves?
	// Or branch the save logic one more time

	// sqlite can handle multiple concurrent reads, writes - not so much
	// make it doubleplusgood certain we're not writing in parallel
	var mutex = &sync.Mutex{}
	mutex.Lock()

	db, err := sql.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	var stmt *sql.Stmt

	// If we are doing partial hashing, save as partial hash
	if partial {
		// using partial hashing, file is smaller than partial limit, save to both full and partial hash (as they will be the same)
		if size < partialSize {
			stmt, err = tx.Prepare("insert into dupes(path, hash, partialhash, date) values(?, ?, ?, CURRENT_TIMESTAMP) on conflict(path) do update set partialhash=?, hash=?, date=CURRENT_TIMESTAMP")

			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()
			_, err = stmt.Exec(filename, hash, hash, hash, hash)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// Partial, save to partialhash
			stmt, err = tx.Prepare("insert into dupes(path, partialhash, date) values(?, ?, CURRENT_TIMESTAMP) on conflict(path) do update set partialhash=?")
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()
			_, err = stmt.Exec(filename, hash, hash)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		// full hash
		stmt, err = tx.Prepare("insert into dupes(path, hash, date) values(?, ?, CURRENT_TIMESTAMP) on conflict(path) do update set hash=?")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		log.Debugf("Inserting: %s - %s\n", filename, hash)
		_, err = stmt.Exec(filename, hash, hash)
		if err != nil {
			log.Fatal(err)
		}

	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	mutex.Unlock()
}
