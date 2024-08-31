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

func IsFileCompleted(db *sql.DB, filePath string, duration time.Duration) bool {
	initialSize, err := GetFileSizeFromDB(db, filePath)
	if err != nil {
		LogWithDatetime("Error getting file size from DB:", err)
		return false
	}

	if initialSize == -1 {
		initialSize = GetFileSize(filePath)
		SaveFileSize(db, filePath, initialSize)
	}

	time.Sleep(duration)
	finalSize := GetFileSize(filePath)
	SaveFileSize(db, filePath, finalSize)

	return initialSize == finalSize
}

func GetFileSize(filePath string) int64 {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return -1
	}
	return fileInfo.Size()
}

func CopyFile(ctx context.Context, src, dst string) (int64, error) {
	dstPart := dst + ".cat.part"
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

	fmt.Printf("\nFinished copying %s to %s\n", src, dst)
	LogWithDatetime(fmt.Sprintf("Finished copying %s to %s", src, dst))
	return totalSize, nil
}

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
