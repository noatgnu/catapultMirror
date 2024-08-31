// catapult/monitor.go
package catapult

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var dbMutex sync.Mutex

func MonitorAndMirror(ctx context.Context, db *sql.DB, config Configurations) {
	InitSlack(config)

	for _, cfg := range config.Configs {
		duration, err := time.ParseDuration(cfg.CheckInterval)
		if err != nil {
			LogWithDatetime("Invalid check_interval:", err)
			sendSlackNotification(fmt.Sprintf("Invalid check_interval: %v", err))
			return
		}
		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		done := make(chan bool)

		go func(cfg Configuration) {
			for {
				select {
				case <-ctx.Done():
					LogWithDatetime("Shutting down monitoring")
					done <- true
					return
				case <-ticker.C:
					LogWithDatetime(fmt.Sprintf("Checking free space for destination: %s", cfg.Destination))
					freeSpace, err := GetFreeSpace(cfg.Destination)
					if err != nil {
						LogWithDatetime("Error getting free space:", err)
						sendSlackNotification(fmt.Sprintf("Error getting free space: %v", err))
						done <- true
						return
					}

					if freeSpace <= cfg.MinFreeSpace {
						LogWithDatetime("No space left at destination. Shutting down gracefully.")
						sendSlackNotification("No space left at destination. Shutting down gracefully.")
						done <- true
						return
					}

					for _, dir := range cfg.Directories {
						LogWithDatetime(fmt.Sprintf("Listing files in directory: %s", dir))
						files, err := ListFiles(dir)
						if err != nil {
							LogWithDatetime("Error listing files:", err)
							sendSlackNotification(fmt.Sprintf("Error listing files: %v", err))
							continue
						}

						for _, file := range files {

							copied, err := IsFileCopied(db, file)
							if err != nil {
								LogWithDatetime("Error checking if file is copied:", err)
								sendSlackNotification(fmt.Sprintf("Error checking if file is copied: %v", err))
								continue
							}
							if copied {
								continue
							}

							if IsFileCompleted(db, file, duration) {
								relPath, err := filepath.Rel(dir, file)
								if err != nil {
									LogWithDatetime("Error getting relative path:", err)
									sendSlackNotification(fmt.Sprintf("Error getting relative path: %v", err))
									continue
								}
								destPath := filepath.Join(cfg.Destination, relPath)

								fileSize := GetFileSize(file)
								if freeSpace-fileSize <= cfg.MinFreeSpace {
									LogWithDatetime("File size will breach minimum free space. Shutting down gracefully.")
									sendSlackNotification("File size will breach minimum free space. Shutting down gracefully.")
									done <- true
									return
								}

								sendSlackNotification(fmt.Sprintf("Starting to copy file: %s", file))
								_, err = CopyFile(ctx, file, destPath)
								if err != nil {
									LogWithDatetime("Error copying file:", err)
									sendSlackNotification(fmt.Sprintf("Error copying file: %v", err))
								} else {
									LogWithDatetime(fmt.Sprintf("Copied file: %s to %s.cat.part", file, destPath))

									originalHash, err := CalculateFileHash(file)
									if err != nil {
										LogWithDatetime("Error calculating hash for original file:", err)
										sendSlackNotification(fmt.Sprintf("Error calculating hash for original file: %v", err))
										continue
									}

									copiedHash, err := CalculateFileHash(destPath + ".cat.part")
									if err != nil {
										LogWithDatetime("Error calculating hash for copied file:", err)
										sendSlackNotification(fmt.Sprintf("Error calculating hash for copied file: %v", err))
										continue
									}

									if originalHash == copiedHash {
										err := os.Rename(destPath+".cat.part", destPath)
										if err != nil {
											LogWithDatetime("Error renaming file:", err)
											sendSlackNotification(fmt.Sprintf("Error renaming file: %v", err))
										} else {
											LogWithDatetime(fmt.Sprintf("File verified and renamed: %s", destPath))
											sendSlackNotification(fmt.Sprintf("Finished copying file: %s", destPath))
											dbMutex.Lock()
											MarkFileAsCopied(db, file)
											dbMutex.Unlock()
										}
									} else {
										LogWithDatetime(fmt.Sprintf("File hash mismatch for: %s", file))
										sendSlackNotification(fmt.Sprintf("File hash mismatch for: %s", file))
										os.Remove(destPath + ".cat.part")
									}
								}
							}
						}
					}
				}
			}
		}(cfg)

		<-done
		LogWithDatetime("Shutting down gracefully")
	}
}
