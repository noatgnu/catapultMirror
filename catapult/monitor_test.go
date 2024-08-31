// catapult/monitor_test.go
package catapult

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestMonitorAndMirror(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a test file in the source directory
	testFilePath := filepath.Join(srcDir, "testfile.txt")
	err := os.WriteFile(testFilePath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := Configuration{
		Directories:   []string{srcDir},
		Destination:   dstDir,
		CheckInterval: "1s",
		MinFreeSpace:  100 * 1024 * 1024, // 100 MB
	}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	go MonitorAndMirror(db, config.Directories, config.Destination, config.CheckInterval, config.MinFreeSpace)

	// Allow some time for the monitor to run
	time.Sleep(2 * time.Second)

	// Check if the file was copied
	copiedFilePath := filepath.Join(dstDir, "testfile.txt")
	if _, err := os.Stat(copiedFilePath); os.IsNotExist(err) {
		t.Fatalf("File was not copied: %v", copiedFilePath)
	}

	// Verify the content of the copied file
	copiedContent, err := os.ReadFile(copiedFilePath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}
	if string(copiedContent) != "test content" {
		t.Fatalf("Copied file content mismatch: got %v, want %v", string(copiedContent), "test content")
	}
}
