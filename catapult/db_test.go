// catapult/db_test.go
package catapult

import (
	"testing"
)

func TestInitDB(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB() error: %v", err)
	}
	defer db.Close()

	// Check if the table was created
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='file_sizes';").Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to query table: %v", err)
	}
	if tableName != "file_sizes" {
		t.Fatalf("Table name = %v, want %v", tableName, "file_sizes")
	}
}

func TestMarkFileAsCopied(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	_, err := db.Exec("INSERT INTO file_sizes (path, size, copied) VALUES (?, ?, ?)", filePath, 100, false)
	if err != nil {
		t.Fatalf("Failed to insert file: %v", err)
	}

	err = MarkFileAsCopied(db, filePath)
	if err != nil {
		t.Fatalf("MarkFileAsCopied() error: %v", err)
	}

	var copied bool
	err = db.QueryRow("SELECT copied FROM file_sizes WHERE path = ?", filePath).Scan(&copied)
	if err != nil {
		t.Fatalf("Failed to query file: %v", err)
	}
	if !copied {
		t.Fatalf("copied = %v, want %v", copied, true)
	}
}

func TestSaveFileSize(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	fileSize := int64(100)

	err := SaveFileSize(db, filePath, fileSize)
	if err != nil {
		t.Fatalf("SaveFileSize() error: %v", err)
	}

	var size int64
	err = db.QueryRow("SELECT size FROM file_sizes WHERE path = ?", filePath).Scan(&size)
	if err != nil {
		t.Fatalf("Failed to query file: %v", err)
	}
	if size != fileSize {
		t.Fatalf("size = %v, want %v", size, fileSize)
	}
}

func TestGetFileSizeFromDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	fileSize := int64(100)
	_, err := db.Exec("INSERT INTO file_sizes (path, size) VALUES (?, ?)", filePath, fileSize)
	if err != nil {
		t.Fatalf("Failed to insert file: %v", err)
	}

	size, err := GetFileSizeFromDB(db, filePath)
	if err != nil {
		t.Fatalf("GetFileSizeFromDB() error: %v", err)
	}
	if size != fileSize {
		t.Fatalf("size = %v, want %v", size, fileSize)
	}
}

func TestIsFileCopied(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	_, err := db.Exec("INSERT INTO file_sizes (path, size, copied) VALUES (?, ?, ?)", filePath, 100, true)
	if err != nil {
		t.Fatalf("Failed to insert file: %v", err)
	}

	copied, err := IsFileCopied(db, filePath)
	if err != nil {
		t.Fatalf("IsFileCopied() error: %v", err)
	}
	if !copied {
		t.Fatalf("copied = %v, want %v", copied, true)
	}
}
