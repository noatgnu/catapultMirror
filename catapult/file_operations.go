// catapult/file_operations.go
package catapult

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ListFiles returns a list of all files and directories in the given root directory and its subdirectories,
// excluding the contents of directories ending with .d.
//
// Parameters:
// - root: The root directory to list files and directories from.
//
// Returns:
// - []string: A slice of file and directory paths.
// - error: An error object if there was an issue listing the files and directories.
func ListFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Include directories ending with .d but skip their contents
		if info.IsDir() && filepath.Ext(info.Name()) == ".d" {
			paths = append(paths, path)
			return filepath.SkipDir
		}
		paths = append(paths, path)
		return nil
	})
	return paths, err
}

// IsFileCompleted checks if a file or directory has completed writing by comparing its size over a specified duration.
//
// Parameters:
// - db: The database connection to track file sizes.
// - path: The path of the file or directory to check.
//
// Returns:
// - bool: True if the size has not changed, indicating the file or directory is completed.
func IsFileCompleted(db *sql.DB, path string, isFolder bool) bool {
	var initialSize int64

	if strings.HasSuffix(path, ".d") {
		initialSize = GetDirectorySize(path)
	} else {
		initialSize = GetFileSize(path)
	}

	if initialSize == -1 {
		initialSize = GetFileSize(path)
		dbMutex.Lock()
		SaveFileSize(db, path, initialSize, isFolder)
		dbMutex.Unlock()
		return false
	}

	var finalSize int64
	if strings.HasSuffix(path, ".d") {
		finalSize = GetDirectorySize(path)
	} else {
		finalSize = GetFileSize(path)
	}

	dbMutex.Lock()
	SaveFileSize(db, path, finalSize, isFolder)
	dbMutex.Unlock()

	return initialSize == finalSize
}

// GetDirectorySize returns the total size of all files in the directory.
//
// Parameters:
// - dirPath: The path of the directory to get the size of.
//
// Returns:
// - int64: The total size of the directory in bytes, or -1 if there was an error.
func GetDirectorySize(dirPath string) int64 {
	var totalSize int64
	err := filepath.Walk(dirPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return -1
	}
	return totalSize
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

	bar := progressbar.NewOptions64(totalSize, progressbar.OptionSetDescription(fmt.Sprintf("Copying %s to %s", src, dst)))

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
			err = bar.Add(n)
			if err != nil {
				return 0, err
			}
			if copiedSize == totalSize {
				break copyLoop
			}
		}
	}

	LogWithDatetime(fmt.Sprintf("Finished copying %s to %s", src, dst), true)
	return totalSize, nil
}

// CalculateFileHash calculates the SHA-256 hash of the file or directory at the given path.
//
// Parameters:
// - filePath: The path of the file or directory to calculate the hash for.
//
// Returns:
// - string: The SHA-256 hash in hexadecimal format.
// - error: An error object if there was an issue calculating the hash.
func CalculateFileHash(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.New()

	if info.IsDir() {
		err := filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fileHash, err := CalculateFileHash(path)
				if err != nil {
					return err
				}
				hash.Write([]byte(fileHash))
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	} else {
		file, err := os.Open(filePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
