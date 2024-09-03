// catapult/file_operations_test.go
package catapult

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestCopyFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	src := "testfile.txt"
	dst := "destination/testfile.txt"
	interval := time.Second

	// Create a test file
	err := os.WriteFile(src, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(src)

	// Save initial file size and last modified time
	SaveFileSize(db, src, int64(len("test content")), false)

	// Wait for the interval duration
	time.Sleep(interval)

	// Copy the file
	_, err = CopyFile(ctx, src, dst)
	if err != nil {
		t.Fatalf("CopyFile() error: %v", err)
	}

	// Rename the copied file
	err = os.Rename(dst+".cat.part", dst)

	// Verify the file was copied
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		fmt.Printf("Error: %v\n", err)
		t.Fatalf("Expected file to be copied to %s", dst)
	}
	defer os.RemoveAll("destination")
}
