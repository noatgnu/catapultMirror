// catapult/db.go
package catapult

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", "file:/foobar?vfs=memdb")
	if err != nil {
		t.Fatalf("sql.Open() error: %v", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS file_sizes (
		path TEXT PRIMARY KEY,
		size INTEGER,
		copied BOOLEAN DEFAULT 0
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	return db
}

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS file_sizes (
        path TEXT PRIMARY KEY,
        size INTEGER,
        copied BOOLEAN DEFAULT 0
    );`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func MarkFileAsCopied(db *sql.DB, filePath string) error {
	updateSQL := `UPDATE file_sizes SET copied = 1 WHERE path = ?;`
	_, err := db.Exec(updateSQL, filePath)
	return err
}

func SaveFileSize(db *sql.DB, filePath string, size int64) error {
	insertSQL := `INSERT OR REPLACE INTO file_sizes (path, size) VALUES (?, ?);`
	_, err := db.Exec(insertSQL, filePath, size)
	return err
}

func GetFileSizeFromDB(db *sql.DB, filePath string) (int64, error) {
	var size int64
	querySQL := `SELECT size FROM file_sizes WHERE path = ?;`
	err := db.QueryRow(querySQL, filePath).Scan(&size)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return size, err
}

func IsFileCopied(db *sql.DB, filePath string) (bool, error) {
	var copied bool
	querySQL := `SELECT copied FROM file_sizes WHERE path = ?;`
	err := db.QueryRow(querySQL, filePath).Scan(&copied)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return copied, err
}
