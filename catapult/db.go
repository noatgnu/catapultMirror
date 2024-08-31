// catapult/db.go
package catapult

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
)

// setupTestDB sets up an in-memory SQLite database for testing purposes.
// It creates the necessary table for storing file sizes and copied status.
//
// Parameters:
// - t: The testing object.
//
// Returns:
// - *sql.DB: The initialized in-memory SQLite database.
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

// InitDB initializes the SQLite database with the given file path.
// It creates the necessary table for storing file sizes and copied status if it does not exist.
//
// Parameters:
// - dbPath: The file path for the SQLite database.
//
// Returns:
// - *sql.DB: The initialized SQLite database.
// - error: An error object if there was an issue initializing the database.
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

// MarkFileAsCopied marks a file as copied in the database.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to mark as copied.
//
// Returns:
// - error: An error object if there was an issue updating the database.
func MarkFileAsCopied(db *sql.DB, filePath string) error {
	updateSQL := `UPDATE file_sizes SET copied = 1 WHERE path = ?;`
	_, err := db.Exec(updateSQL, filePath)
	return err
}

// SaveFileSize saves the size of a file in the database.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to save the size for.
// - size: The size of the file in bytes.
//
// Returns:
// - error: An error object if there was an issue inserting or updating the database.
func SaveFileSize(db *sql.DB, filePath string, size int64) error {
	insertSQL := `INSERT OR REPLACE INTO file_sizes (path, size) VALUES (?, ?);`
	_, err := db.Exec(insertSQL, filePath, size)
	return err
}

// GetFileSizeFromDB retrieves the size of a file from the database.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to retrieve the size for.
//
// Returns:
// - int64: The size of the file in bytes, or -1 if the file size is not found.
// - error: An error object if there was an issue querying the database.
func GetFileSizeFromDB(db *sql.DB, filePath string) (int64, error) {
	var size int64
	querySQL := `SELECT size FROM file_sizes WHERE path = ?;`
	err := db.QueryRow(querySQL, filePath).Scan(&size)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return size, err
}

// IsFileCopied checks if a file is marked as copied in the database.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to check.
//
// Returns:
// - bool: True if the file is marked as copied, false otherwise.
// - error: An error object if there was an issue querying the database.
func IsFileCopied(db *sql.DB, filePath string) (bool, error) {
	var copied bool
	querySQL := `SELECT copied FROM file_sizes WHERE path = ?;`
	err := db.QueryRow(querySQL, filePath).Scan(&copied)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return copied, err
}
