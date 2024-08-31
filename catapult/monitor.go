// catapult/monitor.go
package catapult

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func MonitorAndMirror(db *sql.DB, directories []string, destination string, checkInterval string, minFreeSpace int64) {
	duration, err := time.ParseDuration(checkInterval)
	if err != nil {
		LogWithDatetime("Invalid check_interval:", err)
		return
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-shutdown:
				LogWithDatetime("Received shutdown signal")
				cancel()
				done <- true
				return
			default:
				freeSpace, err := GetFreeSpace(destination)
				if err != nil {
					LogWithDatetime("Error getting free space:", err)
					done <- true
					return
				}

				if freeSpace <= minFreeSpace {
					LogWithDatetime("No space left at destination. Shutting down gracefully.")
					cancel()
					done <- true
					return
				}

				for _, dir := range directories {
					files, err := ListFiles(dir)
					if err != nil {
						LogWithDatetime("Error listing files:", err)
						continue
					}

					for _, file := range files {
						copied, err := IsFileCopied(db, file)
						if err != nil {
							LogWithDatetime("Error checking if file is copied:", err)
							continue
						}
						if copied {
							continue
						}

						if IsFileCompleted(db, file, duration) {
							relPath, err := filepath.Rel(dir, file)
							if err != nil {
								LogWithDatetime("Error getting relative path:", err)
								continue
							}
							destPath := filepath.Join(destination, relPath)

							fileSize := GetFileSize(file)
							if freeSpace-fileSize <= minFreeSpace {
								LogWithDatetime("File size will breach minimum free space. Shutting down gracefully.")
								cancel()
								done <- true
								return
							}

							_, err = CopyFile(ctx, file, destPath)
							if err != nil {
								LogWithDatetime("Error copying file:", err)
							} else {
								LogWithDatetime(fmt.Sprintf("Copied file: %s to %s.cat.part", file, destPath))

								originalHash, err := CalculateFileHash(file)
								if err != nil {
									LogWithDatetime("Error calculating hash for original file:", err)
									continue
								}

								copiedHash, err := CalculateFileHash(destPath + ".cat.part")
								if err != nil {
									LogWithDatetime("Error calculating hash for copied file:", err)
									continue
								}

								if originalHash == copiedHash {
									err := os.Rename(destPath+".cat.part", destPath)
									if err != nil {
										LogWithDatetime("Error renaming file:", err)
									} else {
										LogWithDatetime(fmt.Sprintf("File verified and renamed: %s", destPath))
										MarkFileAsCopied(db, file)
									}
								} else {
									LogWithDatetime(fmt.Sprintf("File hash mismatch for: %s", file))
									os.Remove(destPath + ".cat.part")
								}
							}
						}
					}
				}
				time.Sleep(duration)
			}
		}
	}()

	<-done
	LogWithDatetime("Shutting down gracefully")
}
