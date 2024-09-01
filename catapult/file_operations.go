// catapult/file_operations.go
package catapult

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ListFiles returns a list of all files in the given root directory and its subdirectories.
//
// Parameters:
// - root: The root directory to list files from.
//
// Returns:
// - []string: A slice of file paths.
// - error: An error object if there was an issue listing the files.
func ListFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// IsFileCompleted checks if a file has completed writing by comparing its size over a specified duration.
//
// Parameters:
// - db: The database connection to track file sizes.
// - filePath: The path of the file to check.
// - duration: The duration to wait before checking the file size again.
//
// Returns:
// - bool: True if the file size has not changed, indicating the file is completed.
func IsFileCompleted(db *sql.DB, filePath string, duration time.Duration) bool {
	initialSize, err := GetFileSizeFromDB(db, filePath)
	if err != nil {
		LogWithDatetime(fmt.Sprintf("Error getting file size from DB: %v", err), true)
		return false
	}

	if initialSize == -1 {
		initialSize = GetFileSize(filePath)
		dbMutex.Lock()
		SaveFileSize(db, filePath, initialSize)
		dbMutex.Unlock()
	}

	time.Sleep(duration)
	finalSize := GetFileSize(filePath)
	dbMutex.Lock()
	SaveFileSize(db, filePath, finalSize)
	dbMutex.Unlock()

	return initialSize == finalSize
}

// GetFileSize returns the size of the file at the given path.
//
// Parameters:
// - filePath: The path of the file to get the size of.
//
// Returns:
// - int64: The size of the file in bytes, or -1 if there was an error.
func GetFileSize(filePath string) int64 {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return -1
	}
	return fileInfo.Size()
}

// CopyFile copies a file from the source path to the destination path, with support for context cancellation.
//
// Parameters:
// - ctx: The context to control the file copying lifecycle.
// - src: The source file path.
// - dst: The destination file path.
//
// Returns:
// - int64: The total size of the copied file in bytes.
// - error: An error object if there was an issue copying the file.
func CopyFile(ctx context.Context, src, dst string) (int64, error) {

	dstPart := dst + ".cat.part"
	// check if the file already exists
	if _, err := os.Stat(dstPart); err == nil {
		// file already exists, remove it
		if err := os.Remove(dstPart); err != nil {
			fmt.Printf("Error removing existing file: %v\n", err)
			return 0, err
		}
	}
	sourceFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dstPart), os.ModePerm); err != nil {
		return 0, err
	}

	destinationFile, err := os.Create(dstPart)
	if err != nil {
		return 0, err
	}
	defer destinationFile.Close()

	sourceFileInfo, err := sourceFile.Stat()
	if err != nil {
		return 0, err
	}
	totalSize := sourceFileInfo.Size()
	buffer := make([]byte, 1024*1024)
	var copiedSize int64

copyLoop:
	for {
		select {
		case <-ctx.Done():
			os.Remove(dstPart)
			return 0, ctx.Err()
		default:

			n, err := sourceFile.Read(buffer)
			if err != nil && err != io.EOF {
				return 0, err
			}
			if n == 0 {
				break
			}

			if _, err := destinationFile.Write(buffer[:n]); err != nil {
				return 0, err
			}

			copiedSize += int64(n)
			fmt.Printf("\rCopying %s to %s: %.2f%%", src, dst, float64(copiedSize)/float64(totalSize)*100)
			if copiedSize == totalSize {
				break copyLoop
			}
		}
	}

	LogWithDatetime(fmt.Sprintf("Finished copying %s to %s", src, dst), true)
	return totalSize, nil
}

// CalculateFileHash calculates the SHA-256 hash of the file at the given path.
//
// Parameters:
// - filePath: The path of the file to calculate the hash for.
//
// Returns:
// - string: The SHA-256 hash of the file in hexadecimal format.
// - error: An error object if there was an issue calculating the hash.
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
