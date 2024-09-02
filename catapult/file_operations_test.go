// catapult/file_operations_test.go
package catapult

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileWithVerification_File(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	file := "testfile.txt"
	dir := "."
	destination := "destination"
	cfg := Configuration{}
	freeSpace := int64(1000)

	// Create a test file
	err := os.WriteFile(file, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(file)

	copyFileWithVerification(ctx, db, file, dir, destination, cfg, freeSpace)

	// Verify the file was copied
	destPath := filepath.Join(destination, file)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Fatalf("Expected file to be copied to %s", destPath)
	}
}

func TestCopyFileWithVerification_Folder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	folder := "testfolder"
	dir := "."
	destination := "destination"
	cfg := Configuration{}
	freeSpace := int64(1000)

	// Create a test folder with a file
	err := os.Mkdir(folder, 0755)
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}
	defer os.RemoveAll(folder)

	file := filepath.Join(folder, "testfile.txt")
	err = os.WriteFile(file, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	copyFileWithVerification(ctx, db, folder, dir, destination, cfg, freeSpace)

	// Verify the folder and file were copied
	destFolderPath := filepath.Join(destination, folder)
	if _, err := os.Stat(destFolderPath); os.IsNotExist(err) {
		t.Fatalf("Expected folder to be copied to %s", destFolderPath)
	}

	destFilePath := filepath.Join(destFolderPath, "testfile.txt")
	if _, err := os.Stat(destFilePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to be copied to %s", destFilePath)
	}
}
