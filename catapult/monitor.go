// catapult/monitor.go
package catapult

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var dbMutex sync.Mutex

// MonitorAndMirror initializes the Slack client and starts monitoring directories as per the provided configurations.
// It launches a goroutine for each directory configuration to monitor and mirror files.
//
// Parameters:
// - ctx: The context to control the monitoring lifecycle.
// - db: The database connection to track copied files.
// - config: The configurations containing directory monitoring settings.
func MonitorAndMirror(ctx context.Context, db *sql.DB, config Configurations) {
	InitSlack(config)

	var wg sync.WaitGroup

	for _, cfg := range config.Configs {
		wg.Add(1)
		go func(cfg Configuration) {
			defer wg.Done()
			monitorDirectory(ctx, db, cfg)
		}(cfg)
	}

	wg.Wait()
}

// monitorDirectory monitors a single directory for new files and processes them at specified intervals.
// It checks the free space in the destination directory and processes files if there is enough space.
//
// Parameters:
// - ctx: The context to control the monitoring lifecycle.
// - db: The database connection to track copied files.
// - cfg: The configuration for the directory to monitor.
func monitorDirectory(ctx context.Context, db *sql.DB, cfg Configuration) {
	duration, err := time.ParseDuration(cfg.CheckInterval)
	if err != nil {
		LogWithDatetime("Invalid check_interval:", true)
		sendSlackNotification(fmt.Sprintf("Invalid check_interval: %v", err))
		return
	}
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			LogWithDatetime("Shutting down monitoring", true)
			return
		case <-ticker.C:

			for _, destination := range cfg.Destinations {
				fmt.Printf("Monitoring directory: %s\n", cfg.Name)
				LogWithDatetime(fmt.Sprintf("Checking free space for destination: %s", destination), true)
				freeSpace, err := GetFreeSpace(destination)
				if err != nil {
					LogWithDatetime("Error getting free space:", true)
					sendSlackNotification(fmt.Sprintf("Error getting free space: %v", err))
					return
				}

				if freeSpace <= cfg.MinFreeSpace {
					LogWithDatetime("No space left at destination. Shutting down gracefully.", true)
					sendSlackNotification("No space left at destination. Shutting down gracefully.")
					return
				}

				for _, dir := range cfg.Directories {
					fmt.Printf("Processing directory: %s\n", dir)
					processFiles(ctx, db, dir, cfg, destination, freeSpace, duration)
				}
			}
		}
	}
}

// processFiles processes the files and directories in a directory, checking if they are completed and copying them if necessary.
// It verifies if the files and directories are already copied and if they are completed before initiating the copy process.
//
// Parameters:
// - ctx: The context to control the file processing lifecycle.
// - db: The database connection to track copied files.
// - dir: The directory to process files and directories from.
// - cfg: The configuration for the directory to monitor.
// - freeSpace: The available free space in the destination directory.
// - duration: The interval duration for checking file completion.
// processFiles processes the files and directories in a directory, checking if they are completed and copying them if necessary.
// It verifies if the files and directories are already copied and if they are completed before initiating the copy process.
//
// Parameters:
// - ctx: The context to control the file processing lifecycle.
// - db: The database connection to track copied files.
// - dir: The directory to process files and directories from.
// - cfg: The configuration for the directory to monitor.
// - freeSpace: The available free space in the destination directory.
// - duration: The interval duration for checking file completion.
func processFiles(ctx context.Context, db *sql.DB, dir string, cfg Configuration, destination string, freeSpace int64, duration time.Duration) {
	LogWithDatetime(fmt.Sprintf("Listing files and directories in directory: %s", dir), true)
	paths, err := ListFiles(dir)
	if err != nil {
		LogWithDatetime("Error listing files and directories:", true)
		sendSlackNotification(fmt.Sprintf("Error listing files and directories: %v", err))
		return
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			LogWithDatetime(fmt.Sprintf("Error stating path: %v", err), true)
			sendSlackNotification(fmt.Sprintf("Error stating path: %v", err))
			continue
		}
		isFolder := info.IsDir()
		var size int64
		if isFolder && strings.HasSuffix(path, ".d") {
			size = GetDirectorySize(path)
		} else if !isFolder {
			size = GetFileSize(path)
		} else {
			continue
		}

		// Ignore files or directories smaller than the minimum file size
		if size < cfg.MinFileSize {
			LogWithDatetime(fmt.Sprintf("Ignoring file or directory smaller than minimum size: %s", path), false)
			continue
		}
		// Ignore empty files or directories
		if size == 0 {
			LogWithDatetime(fmt.Sprintf("Ignoring empty file or directory: %s", path), false)
			continue
		}

		initialSize, err := GetFileSizeFromDB(db, path, isFolder)
		if err != nil {
			LogWithDatetime(fmt.Sprintf("Error getting file size from DB: %v", err), true)
			sendSlackNotification(fmt.Sprintf("Error getting file size from DB: %v", err))
			continue
		}

		if initialSize == -1 {
			dbMutex.Lock()
			SaveFileSize(db, path, size, isFolder)
			dbMutex.Unlock()
			LogWithDatetime(fmt.Sprintf("First time seeing file or directory, added to DB: %s", path), false)
			continue
		}

		if initialSize != size {
			dbMutex.Lock()
			SaveFileSize(db, path, size, isFolder)
			dbMutex.Unlock()
			LogWithDatetime(fmt.Sprintf("File or directory size changed, not ready for copying: %s", path), false)
			continue
		}

		lastModified, err := GetLastModifiedTime(db, path)
		if err != nil {
			LogWithDatetime(fmt.Sprintf("Error retrieving last modified time: %v", err), true)
			sendSlackNotification(fmt.Sprintf("Error retrieving last modified time: %v", err))
			continue
		}

		if time.Since(lastModified) < duration {
			LogWithDatetime(fmt.Sprintf("File or directory not ready for copying due to recent modification: %s", path), false)
			continue
		}

		copied, err := IsFileCopied(db, path, destination, isFolder)
		if err != nil {
			LogWithDatetime(fmt.Sprintf("Error checking if file or directory is copied: %v", err), true)
			sendSlackNotification(fmt.Sprintf("Error checking if file or directory is copied: %v", err))
			continue
		}
		if copied {
			continue
		}

		copyFileWithVerification(ctx, db, path, dir, destination, cfg, freeSpace)
	}
}

// copyFileWithVerification copies a file or directory to the destination directory and verifies its integrity by comparing file hashes.
// If the file hashes match, the file is renamed to its final destination name.
//
// Parameters:
// - ctx: The context to control the file copying lifecycle.
// - db: The database connection to track copied files.
// - file: The file or directory to be copied.
// - dir: The source directory of the file or directory.
// - cfg: The configuration for the directory to monitor.
// - freeSpace: The available free space in the destination directory.
func copyFileWithVerification(ctx context.Context, db *sql.DB, file, dir, destination string, cfg Configuration, freeSpace int64) {
	relPath, err := filepath.Rel(dir, file)
	if err != nil {
		LogWithDatetime(fmt.Sprintf("Error getting relative path: %v", err), true)
		sendSlackNotification(fmt.Sprintf("Error getting relative path: %v", err))
		return
	}
	destPath := filepath.Join(destination, relPath)

	info, err := os.Stat(file)
	if err != nil {
		LogWithDatetime(fmt.Sprintf("Error stating path: %v", err), true)
		sendSlackNotification(fmt.Sprintf("Error stating path: %v", err))
		return
	}
	isFolder := info.IsDir()

	if isFolder {
		LogWithDatetime(fmt.Sprintf("Starting to copy folder: `%s` to destination: `%s`", file, destPath), true)
		sendSlackNotification(fmt.Sprintf("Starting to copy folder: `%s` to destination: `%s`", file, destPath))
		err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(file, path)
			if err != nil {
				return err
			}
			destFilePath := filepath.Join(destPath, relPath)

			if info.IsDir() {
				if err := os.MkdirAll(destFilePath, os.ModePerm); err != nil {
					return err
				}
			} else {
				if _, err := CopyFile(ctx, path, destFilePath); err != nil {
					return err
				}

				originalHash, err := CalculateFileHash(path)
				if err != nil {
					return err
				}

				copiedHash, err := CalculateFileHash(destFilePath)
				if err != nil {
					return err
				}

				if originalHash != copiedHash {
					return fmt.Errorf("file hash mismatch for: %s", path)
				}
			}
			return nil
		})
		if err != nil {
			LogWithDatetime(fmt.Sprintf("Error copying directory: %v", err), true)
			sendSlackNotification(fmt.Sprintf("Error copying directory: %v", err))
			return
		}
		LogWithDatetime(fmt.Sprintf("Finished copying folder: `%s` to destination: `%s`", file, destPath), true)
		sendSlackNotification(fmt.Sprintf("Finished copying folder: `%s` to destination: `%s`", file, destPath))
	} else {
		// Check if the file already exists at the destination
		if _, err := os.Stat(destPath); err == nil {
			// File exists, calculate hashes
			originalHash, err := CalculateFileHash(file)
			if err != nil {
				LogWithDatetime(fmt.Sprintf("Error calculating hash for original file: %v", err), true)
				sendSlackNotification(fmt.Sprintf("Error calculating hash for original file: %v", err))
				return
			}

			destinationHash, err := CalculateFileHash(destPath)
			if err != nil {
				LogWithDatetime(fmt.Sprintf("Error calculating hash for destination file: %v", err), true)
				sendSlackNotification(fmt.Sprintf("Error calculating hash for destination file: %v", err))
				return
			}

			if originalHash == destinationHash {
				LogWithDatetime(fmt.Sprintf("File already exists and is identical: %s", destPath), true)
				sendSlackNotification(fmt.Sprintf("File already exists and is identical: %s", destPath))
				dbMutex.Lock()
				MarkFileAsCopied(db, file, destination, isFolder)
				dbMutex.Unlock()
				return
			} else {
				LogWithDatetime(fmt.Sprintf("File already exists but is different: %s", destPath), true)
				sendSlackNotification(fmt.Sprintf("File already exists but is different: %s", destPath))
				return
			}
		}

		fileSize := GetFileSize(file)
		if freeSpace-fileSize <= cfg.MinFreeSpace {
			LogWithDatetime("File size will breach minimum free space. Shutting down gracefully.", false)
			sendSlackNotification("File size will breach minimum free space. Shutting down gracefully.")
			return
		}

		sendSlackNotification(fmt.Sprintf("Starting to copy file: `%s` to destination: `%s`", file, destPath))
		_, err = CopyFile(ctx, file, destPath)
		if err != nil {
			LogWithDatetime(fmt.Sprintf("Error copying file: %v", err), true)
			sendSlackNotification(fmt.Sprintf("Error copying file: %v", err))
		} else {
			LogWithDatetime(fmt.Sprintf("Copied file: %s to %s.cat.part", file, destPath), true)

			originalHash, err := CalculateFileHash(file)
			if err != nil {
				LogWithDatetime(fmt.Sprintf("Error calculating hash for original file: %v", err), true)
				sendSlackNotification(fmt.Sprintf("Error calculating hash for original file: %v", err))
				return
			}

			copiedHash, err := CalculateFileHash(destPath + ".cat.part")
			if err != nil {
				LogWithDatetime(fmt.Sprintf("Error calculating hash for copied file: %v", err), true)
				sendSlackNotification(fmt.Sprintf("Error calculating hash for copied file: %v", err))
				return
			}

			if originalHash == copiedHash {
				err := os.Rename(destPath+".cat.part", destPath)
				if err != nil {
					LogWithDatetime(fmt.Sprintf("Error renaming file: %v", err), true)
					sendSlackNotification(fmt.Sprintf("Error renaming file: %v", err))
				} else {
					LogWithDatetime(fmt.Sprintf("File verified and renamed: %s", destPath), true)
					sendSlackNotification(fmt.Sprintf("Finished copying file: %s", destPath))
					dbMutex.Lock()
					MarkFileAsCopied(db, file, destination, isFolder)
					dbMutex.Unlock()
				}
			} else {
				LogWithDatetime(fmt.Sprintf("File hash mismatch for: %s", file), true)
				sendSlackNotification(fmt.Sprintf("File hash mismatch for: %s", file))
				os.Remove(destPath + ".cat.part")
			}
		}
	}
}
