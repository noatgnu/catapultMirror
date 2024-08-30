package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

var logFile *os.File

// Configuration struct to hold the JSON configuration
type Configuration struct {
	Directories   []string `json:"directories"`
	Destination   string   `json:"destination"`
	CheckInterval string   `json:"check_interval"`
	MinFreeSpace  int64    `json:"min_free_space"` // New field for minimum free space
}

// Create a template configuration file
func createTemplateConfig(filePath string) error {
	templateConfig := Configuration{
		Directories:   []string{"exampleDir1", "exampleDir2"},
		Destination:   "exampleDestinationDir",
		CheckInterval: "1m",
		MinFreeSpace:  10000 * 1024 * 1024, // 10 GB
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(templateConfig)
}

// Initialize the SQLite database
func initDB(dbPath string) (*sql.DB, error) {
	// Create the database file if it does not exist
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create the table if it does not exist
	createTableSQL := `CREATE TABLE IF NOT EXISTS file_sizes (
		path TEXT PRIMARY KEY,
		size INTEGER,
		copied BOOLEAN DEFAULT 0
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Mark a file as copied in the database
func markFileAsCopied(db *sql.DB, filePath string) error {
	updateSQL := `UPDATE file_sizes SET copied = 1 WHERE path = ?;`
	_, err := db.Exec(updateSQL, filePath)
	return err
}

// Save file size to the database
func saveFileSize(db *sql.DB, filePath string, size int64) error {
	insertSQL := `INSERT OR REPLACE INTO file_sizes (path, size) VALUES (?, ?);`
	_, err := db.Exec(insertSQL, filePath, size)
	return err
}

// Get file size from the database
func getFileSizeFromDB(db *sql.DB, filePath string) (int64, error) {
	var size int64
	querySQL := `SELECT size FROM file_sizes WHERE path = ?;`
	err := db.QueryRow(querySQL, filePath).Scan(&size)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return size, err
}

// List all files in a directory and its subdirectories
func listFiles(root string) ([]string, error) {
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

// Check if a file is completed by monitoring its size
func isFileCompleted(db *sql.DB, filePath string, duration time.Duration) bool {
	initialSize, err := getFileSizeFromDB(db, filePath)
	if err != nil {
		logWithDatetime("Error getting file size from DB:", err)
		return false
	}

	if initialSize == -1 {
		initialSize = getFileSize(filePath)
		saveFileSize(db, filePath, initialSize)
	}

	time.Sleep(duration)
	finalSize := getFileSize(filePath)
	saveFileSize(db, filePath, finalSize)

	return initialSize == finalSize
}

// Get the size of a file
func getFileSize(filePath string) int64 {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return -1
	}
	return fileInfo.Size()
}

// Copy a file to a specified location with ".cat.part" suffix and display progress
func copyFile(ctx context.Context, src, dst string) (int64, error) {
	dstPart := dst + ".cat.part"
	sourceFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sourceFile.Close()

	// Create the necessary directories in the destination path
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
	buffer := make([]byte, 1024*1024) // 1MB buffer
	var copiedSize int64

	for {
		select {
		case <-ctx.Done():
			// Delete the .cat.part file if the context is cancelled
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
			// Skip logging transfer progress
		}
	}

	logWithDatetime(fmt.Sprintf("Finished copying %s to %s", src, dst))
	return totalSize, nil
}

// Calculate the SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
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

// Check if a file has been copied
func isFileCopied(db *sql.DB, filePath string) (bool, error) {
	var copied bool
	querySQL := `SELECT copied FROM file_sizes WHERE path = ?;`
	err := db.QueryRow(querySQL, filePath).Scan(&copied)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return copied, err
}

// Monitor directories and mirror completed files
func monitorAndMirror(db *sql.DB, directories []string, destination string, checkInterval string, minFreeSpace int64) {
	duration, err := time.ParseDuration(checkInterval)
	if err != nil {
		logWithDatetime("Invalid check_interval:", err)
		return
	}

	// Channel to listen for OS signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Channel to signal the end of the monitoring loop
	done := make(chan bool)

	// Context to manage cancellation
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-shutdown:
				logWithDatetime("Received shutdown signal")
				cancel() // Cancel the context to stop any ongoing file copy
				done <- true
				return
			default:
				freeSpace, err := getFreeSpace(destination)
				if err != nil {
					logWithDatetime("Error getting free space:", err)
					done <- true
					return
				}

				if freeSpace <= minFreeSpace {
					logWithDatetime("No space left at destination. Shutting down gracefully.")
					cancel() // Cancel the context to stop any ongoing file copy
					done <- true
					return
				}

				for _, dir := range directories {
					files, err := listFiles(dir)
					if err != nil {
						logWithDatetime("Error listing files:", err)
						continue
					}

					for _, file := range files {
						copied, err := isFileCopied(db, file)
						if err != nil {
							logWithDatetime("Error checking if file is copied:", err)
							continue
						}
						if copied {
							continue
						}

						if isFileCompleted(db, file, duration) {
							relPath, err := filepath.Rel(dir, file)
							if err != nil {
								logWithDatetime("Error getting relative path:", err)
								continue
							}
							destPath := filepath.Join(destination, relPath)

							// Check if the file size will breach the minimum free space
							fileSize := getFileSize(file)
							if freeSpace-fileSize <= minFreeSpace {
								logWithDatetime("File size will breach minimum free space. Shutting down gracefully.")
								cancel() // Cancel the context to stop any ongoing file copy
								done <- true
								return
							}

							_, err = copyFile(ctx, file, destPath)
							if err != nil {
								logWithDatetime("Error copying file:", err)
							} else {
								logWithDatetime(fmt.Sprintf("Copied file: %s to %s.cat.part", file, destPath))

								// Verify SHA-256 hash
								originalHash, err := calculateFileHash(file)
								if err != nil {
									logWithDatetime("Error calculating hash for original file:", err)
									continue
								}

								copiedHash, err := calculateFileHash(destPath + ".cat.part")
								if err != nil {
									logWithDatetime("Error calculating hash for copied file:", err)
									continue
								}

								if originalHash == copiedHash {
									err := os.Rename(destPath+".cat.part", destPath)
									if err != nil {
										logWithDatetime("Error renaming file:", err)
									} else {
										logWithDatetime(fmt.Sprintf("File verified and renamed: %s", destPath))
										markFileAsCopied(db, file)
									}
								} else {
									logWithDatetime(fmt.Sprintf("File hash mismatch for: %s", file))
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
	logWithDatetime("Shutting down gracefully")
}

// Read configuration from a JSON file
func readConfigFromFile(filePath string) (Configuration, error) {
	var config Configuration
	file, err := os.Open(filePath)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	// Parse the check_interval string into a time.Duration
	duration, err := time.ParseDuration(config.CheckInterval)
	if err != nil {
		return config, fmt.Errorf("invalid check_interval: %v", err)
	}

	config.CheckInterval = duration.String()
	return config, nil
}

// Log messages with datetime
func logWithDatetime(v ...interface{}) {
	message := fmt.Sprintln(append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, v...)...)
	fmt.Print(message)
	if logFile != nil {
		logFile.WriteString(message)
	}
}

func main() {
	configFile := flag.String("config", "", "Path to the JSON configuration file")
	dbPath := flag.String("db", "file_sizes.db", "Path to the SQLite database file")
	logFilePath := flag.String("log", "transfer.log", "Path to the log file")
	flag.Parse()

	var err error
	logFile, err = os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()

	if *configFile == "" {
		logWithDatetime("Usage: catapultMirror -config=<config_file> -db=<db_file> -log=<log_file>")
		return
	}

	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		err := createTemplateConfig(*configFile)
		if err != nil {
			logWithDatetime("Error creating template configuration file:", err)
			return
		}
		logWithDatetime(fmt.Sprintf("Template configuration file created at %s. Please fill in the file and start again.", *configFile))
		return
	}

	config, err := readConfigFromFile(*configFile)
	if err != nil {
		logWithDatetime("Error reading configuration file:", err)
		return
	}

	db, err := initDB(*dbPath)
	if err != nil {
		logWithDatetime("Error initializing database:", err)
		return
	}
	defer db.Close()

	// Log current destination free space in MB
	freeSpace, err := getFreeSpace(config.Destination)
	if err != nil {
		logWithDatetime("Error getting free space:", err)
		return
	}

	logWithDatetime(fmt.Sprintf("Destination free space: %.2f MB", float64(freeSpace)/1024/1024))

	monitorAndMirror(db, config.Directories, config.Destination, config.CheckInterval, config.MinFreeSpace)
}
