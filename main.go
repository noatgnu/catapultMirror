package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Configuration struct to hold the JSON configuration
type Configuration struct {
	Directories   []string      `json:"directories"`
	Destination   string        `json:"destination"`
	CheckInterval time.Duration `json:"check_interval"`
}

// Create a template configuration file
func createTemplateConfig(filePath string) error {
	templateConfig := Configuration{
		Directories:   []string{"exampleDir1", "exampleDir2"},
		Destination:   "exampleDestinationDir",
		CheckInterval: 1 * time.Minute,
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
	db, err := sql.Open("sqlite3", dbPath)
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
		fmt.Println("Error getting file size from DB:", err)
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
func copyFile(src, dst string) error {
	dstPart := dst + ".cat.part"
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dstPart)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	sourceFileInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	totalSize := sourceFileInfo.Size()
	buffer := make([]byte, 1024*1024) // 1MB buffer
	var copiedSize int64

	for {
		n, err := sourceFile.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destinationFile.Write(buffer[:n]); err != nil {
			return err
		}

		copiedSize += int64(n)
		progress := float64(copiedSize) / float64(totalSize) * 100
		fmt.Printf("\rCopying %s: %.2f%% complete", src, progress)
	}

	fmt.Println() // New line after progress is complete
	return nil
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
// Monitor directories and mirror completed files
func monitorAndMirror(db *sql.DB, directories []string, destination string, checkInterval time.Duration) {
	for {
		for _, dir := range directories {
			files, err := listFiles(dir)
			if err != nil {
				fmt.Println("Error listing files:", err)
				continue
			}

			for _, file := range files {
				copied, err := isFileCopied(db, file)
				if err != nil {
					fmt.Println("Error checking if file is copied:", err)
					continue
				}
				if copied {
					continue
				}

				if isFileCompleted(db, file, checkInterval) {
					destPath := filepath.Join(destination, filepath.Base(file))
					err := copyFile(file, destPath)
					if err != nil {
						fmt.Println("Error copying file:", err)
					} else {
						fmt.Println("Copied file:", file, "to", destPath+".cat.part")

						// Verify SHA-256 hash
						originalHash, err := calculateFileHash(file)
						if err != nil {
							fmt.Println("Error calculating hash for original file:", err)
							continue
						}

						copiedHash, err := calculateFileHash(destPath + ".cat.part")
						if err != nil {
							fmt.Println("Error calculating hash for copied file:", err)
							continue
						}

						if originalHash == copiedHash {
							err := os.Rename(destPath+".cat.part", destPath)
							if err != nil {
								fmt.Println("Error renaming file:", err)
							} else {
								fmt.Println("File verified and renamed:", destPath)
								markFileAsCopied(db, file)
							}
						} else {
							fmt.Println("File hash mismatch for:", file)
							os.Remove(destPath + ".cat.part")
						}
					}
				}
			}
		}
		time.Sleep(checkInterval)
	}
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
	return config, err
}

func main() {
	configFile := flag.String("config", "", "Path to the JSON configuration file")
	dbPath := flag.String("db", "file_sizes.db", "Path to the SQLite database file")
	flag.Parse()

	if *configFile == "" {
		fmt.Println("Usage: catapultMirror -config=<config_file> -db=<db_file>")
		return
	}

	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		err := createTemplateConfig(*configFile)
		if err != nil {
			fmt.Println("Error creating template configuration file:", err)
			return
		}
		fmt.Printf("Template configuration file created at %s. Please fill in the file and start again.\n", *configFile)
		return
	}

	config, err := readConfigFromFile(*configFile)
	if err != nil {
		fmt.Println("Error reading configuration file:", err)
		return
	}

	db, err := initDB(*dbPath)
	if err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}
	defer db.Close()

	monitorAndMirror(db, config.Directories, config.Destination, config.CheckInterval)
}
