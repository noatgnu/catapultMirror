// catapult/db.go
package catapult

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
	"time"
)

// setupTestDB sets up an in-memory SQLite database for testing purposes.
// It creates the necessary tables for storing file sizes and copied status.
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

	createTableSQL := `
	 CREATE TABLE IF NOT EXISTS file_sizes (
	  path TEXT PRIMARY KEY,
	  size INTEGER,
	  is_folder BOOLEAN,
	  last_modified TIMESTAMP,
	  checksum TEXT
	 );
	 CREATE TABLE IF NOT EXISTS copied_files (
	  file_path TEXT,
	  destination TEXT,
	  is_folder BOOLEAN,
	  checksum TEXT,
	  size INTEGER,
	  PRIMARY KEY (file_path, destination, is_folder)
	 );`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db
}

// InitDB initializes the SQLite database with the given file path.
// It creates the necessary tables for storing file sizes and copied status if they do not exist.
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

	createTableSQL := `
	 CREATE TABLE IF NOT EXISTS file_sizes (
	  path TEXT PRIMARY KEY,
	  size INTEGER,
	  is_folder BOOLEAN,
	  last_modified TIMESTAMP,
	  checksum TEXT
	 );
	 CREATE TABLE IF NOT EXISTS copied_files (
	  file_path TEXT,
	  destination TEXT,
	  is_folder BOOLEAN,
	  checksum TEXT,
	  size INTEGER,
	  PRIMARY KEY (file_path, destination, is_folder)
 	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// SaveFileSize saves the size of a file in the database.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to save the size for.
// - size: The size of the file in bytes.
// - isFolder: Boolean indicating if the path is a folder.
//
// Returns:
// - error: An error object if there was an issue inserting or updating the database.
func SaveFileSize(db *sql.DB, filePath string, size int64, isFolder bool) error {
	lastModified := time.Now()
	insertSQL := `INSERT OR REPLACE INTO file_sizes (path, size, is_folder, last_modified) VALUES (?, ?, ?, ?);`
	_, err := db.Exec(insertSQL, filePath, size, isFolder, lastModified)
	return err
}

// GetLastModifiedTime retrieves the last modified time of a file or directory from the database.
//
// Parameters:
// - db: The database connection to retrieve the last modified time.
// - path: The path of the file or directory.
//
// Returns:
// - time.Time: The last modified time.
// - error: An error object if there was an issue retrieving the last modified time.
func GetLastModifiedTime(db *sql.DB, path string) (time.Time, error) {
	var lastModified time.Time
	err := db.QueryRow(`SELECT last_modified FROM file_sizes WHERE path = ?`, path).Scan(&lastModified)
	if err != nil {
		return time.Time{}, err
	}
	return lastModified, nil
}

// GetFileSizeFromDB retrieves the size of a file from the database.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to retrieve the size for.
// - isFolder: Boolean indicating if the path is a folder.
//
// Returns:
// - int64: The size of the file in bytes, or -1 if the file size is not found.
// - error: An error object if there was an issue querying the database.
func GetFileSizeFromDB(db *sql.DB, filePath string, isFolder bool) (int64, error) {
	var size int64
	querySQL := `SELECT size FROM file_sizes WHERE path = ? AND is_folder = ?;`
	err := db.QueryRow(querySQL, filePath, isFolder).Scan(&size)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return size, err
}

// IsFileCopied checks if a file has been copied to a specific destination.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to check.
// - destination: The destination to check.
// - isFolder: Boolean indicating if the path is a folder.
//
// Returns:
// - bool: True if the file has been copied to the destination, false otherwise.
// - error: An error object if there was an issue querying the database.
func IsFileCopied(db *sql.DB, filePath, destination string, isFolder bool) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM copied_files WHERE file_path = ? AND destination = ? AND is_folder = ?", filePath, destination, isFolder).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkFileAsCopied marks a file as copied to a specific destination in the database.
//
// Parameters:
// - db: The database connection.
// - filePath: The path of the file to mark as copied.
// - destination: The destination to mark.
// - isFolder: Boolean indicating if the path is a folder.
//
// Returns:
// - error: An error object if there was an issue inserting into the database.
func MarkFileAsCopied(db *sql.DB, filePath, destination string, isFolder bool) error {
	_, err := db.Exec("INSERT INTO copied_files (file_path, destination, is_folder) VALUES (?, ?, ?)", filePath, destination, isFolder)
	return err
}

func UpdateFileChecksum(db *sql.DB, filePath, checksum string) error {
	query := `UPDATE file_sizes SET checksum = ? WHERE path = ?`
	_, err := db.Exec(query, checksum, filePath)
	return err
}

func UpdateCopiedFileChecksum(db *sql.DB, filePath, destination, checksum string) error {
	query := `UPDATE copied_files SET checksum = ? WHERE file_path = ? AND destination = ?`
	_, err := db.Exec(query, checksum, filePath, destination)
	return err
}

func GetOriginFileChecksum(db *sql.DB, filePath string) (string, error) {
	var checksum sql.NullString
	query := `SELECT checksum FROM file_sizes WHERE path = ?`
	err := db.QueryRow(query, filePath).Scan(&checksum)
	if err != nil {
		return "", err
	}
	if checksum.Valid {
		return checksum.String, nil
	}
	return "", nil
}

func GetCopiedFileChecksum(db *sql.DB, filePath, destination string) (string, error) {
	var checksum sql.NullString
	query := `SELECT checksum FROM copied_files WHERE file_path = ? AND destination = ?`
	err := db.QueryRow(query, filePath, destination).Scan(&checksum)
	if err != nil {
		return "", err
	}
	if checksum.Valid {
		return checksum.String, nil
	}
	return "", nil
}

func UpdateCopiedFileSize(db *sql.DB, filePath, destination string, size int64) error {
	query := `UPDATE copied_files SET size = ? WHERE file_path = ? AND destination = ?`
	_, err := db.Exec(query, size, filePath, destination)
	return err
}

func GetCopiedFileSize(db *sql.DB, filePath, destination string) (int64, error) {
	var size int64
	query := `SELECT size FROM copied_files WHERE file_path = ? AND destination = ?`
	err := db.QueryRow(query, filePath, destination).Scan(&size)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return size, err
}
