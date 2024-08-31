// catapult/file_operations_test.go
package catapult

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListFiles(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	files, err := ListFiles(testDir)
	if err != nil {
		t.Fatalf("ListFiles() error: %v", err)
	}

	if len(files) != 1 || files[0] != testFile {
		t.Fatalf("ListFiles() = %v, want %v", files, []string{testFile})
	}
}

func TestIsFileCompleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testFile := "testfile.txt"
	os.WriteFile(testFile, []byte("test content"), 0644)
	defer os.Remove(testFile)

	completed := IsFileCompleted(db, testFile, 1*time.Second)
	if !completed {
		t.Fatalf("IsFileCompleted() = %v, want %v", completed, true)
	}
}

func TestGetFileSize(t *testing.T) {
	testFile := "testfile.txt"
	os.WriteFile(testFile, []byte("test content"), 0644)
	defer os.Remove(testFile)

	size := GetFileSize(testFile)
	if size != int64(len("test content")) {
		t.Fatalf("GetFileSize() = %v, want %v", size, len("test content"))
	}
}

func TestCopyFile(t *testing.T) {
	srcFile := "srcfile.txt"
	dstFile := "dstfile.txt"
	os.WriteFile(srcFile, []byte("test content"), 0644)
	defer os.Remove(srcFile)
	defer os.Remove(dstFile + ".cat.part")

	ctx := context.Background()
	_, err := CopyFile(ctx, srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyFile() error: %v", err)
	}

	srcContent, _ := os.ReadFile(srcFile)
	dstContent, _ := os.ReadFile(dstFile + ".cat.part")
	if string(srcContent) != string(dstContent) {
		t.Fatalf("CopyFile() content mismatch: got %v, want %v", string(dstContent), string(srcContent))
	}
}

func TestCalculateFileHash(t *testing.T) {
	// Create a temporary file with known content
	testFile := "testfile.txt"
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Calculate the expected hash
	hash := sha256.New()
	hash.Write(content)
	expectedHash := hex.EncodeToString(hash.Sum(nil))

	// Call the CalculateFileHash function
	calculatedHash, err := CalculateFileHash(testFile)
	if err != nil {
		t.Fatalf("CalculateFileHash() error: %v", err)
	}

	// Compare the result with the expected hash
	if calculatedHash != expectedHash {
		t.Fatalf("CalculateFileHash() = %v, want %v", calculatedHash, expectedHash)
	}
}
