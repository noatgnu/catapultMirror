// catapult/db_test.go
package catapult

import (
	"testing"
	"time"
)

func TestSaveFileSize(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	fileSize := int64(100)
	isFolder := false

	err := SaveFileSize(db, filePath, fileSize, isFolder)
	if err != nil {
		t.Fatalf("SaveFileSize() error: %v", err)
	}

	var size int64
	err = db.QueryRow("SELECT size FROM file_sizes WHERE path = ? AND is_folder = ?", filePath, isFolder).Scan(&size)
	if err != nil {
		t.Fatalf("Failed to query file: %v", err)
	}
	if size != fileSize {
		t.Fatalf("size = %v, want %v", size, fileSize)
	}
}

func TestGetLastModifiedTime(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	fileSize := int64(100)
	isFolder := false
	lastModified := time.Now()

	_, err := db.Exec("INSERT INTO file_sizes (path, size, is_folder, last_modified) VALUES (?, ?, ?, ?)", filePath, fileSize, isFolder, lastModified)
	if err != nil {
		t.Fatalf("Failed to insert file: %v", err)
	}

	retrievedTime, err := GetLastModifiedTime(db, filePath)
	if err != nil {
		t.Fatalf("GetLastModifiedTime() error: %v", err)
	}
	if !retrievedTime.Equal(lastModified) {
		t.Fatalf("lastModified = %v, want %v", retrievedTime, lastModified)
	}
}

func TestGetFileSizeFromDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	fileSize := int64(100)
	isFolder := false

	_, err := db.Exec("INSERT INTO file_sizes (path, size, is_folder) VALUES (?, ?, ?)", filePath, fileSize, isFolder)
	if err != nil {
		t.Fatalf("Failed to insert file: %v", err)
	}

	size, err := GetFileSizeFromDB(db, filePath, isFolder)
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
	destination := "destination"
	isFolder := false

	_, err := db.Exec("INSERT INTO copied_files (file_path, destination, is_folder) VALUES (?, ?, ?)", filePath, destination, isFolder)
	if err != nil {
		t.Fatalf("Failed to insert copied file: %v", err)
	}

	copied, err := IsFileCopied(db, filePath, destination, isFolder)
	if err != nil {
		t.Fatalf("IsFileCopied() error: %v", err)
	}
	if !copied {
		t.Fatalf("Expected file to be marked as copied")
	}
}

func TestMarkFileAsCopied(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	filePath := "testfile.txt"
	destination := "destination"
	isFolder := false

	err := MarkFileAsCopied(db, filePath, destination, isFolder)
	if err != nil {
		t.Fatalf("MarkFileAsCopied() error: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM copied_files WHERE file_path = ? AND destination = ? AND is_folder = ?", filePath, destination, isFolder).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query copied_files: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected 1 record in copied_files, got %d", count)
	}
}
