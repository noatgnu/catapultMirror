// catapult/monitor_test.go
package catapult

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

/*
func TestMonitorAndMirrorSync(t *testing.T) {

		// Define fixed paths for source and destination directories
		srcDir1 := "test_src1"
		dstDir1 := "test_dst1"
		srcDir2 := "test_src2"
		dstDir2 := "test_dst2"

		// Create source and destination directories
		if err := os.MkdirAll(srcDir1, 0755); err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}
		defer os.RemoveAll(srcDir1)

		if err := os.MkdirAll(dstDir1, 0755); err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}
		defer os.RemoveAll(dstDir1)

		if err := os.MkdirAll(srcDir2, 0755); err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}
		defer os.RemoveAll(srcDir2)

		if err := os.MkdirAll(dstDir2, 0755); err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}
		defer os.RemoveAll(dstDir2)

		// Create test files in the source directories
		testFilePath1 := filepath.Join(srcDir1, "testfile1.txt")
		err := os.WriteFile(testFilePath1, []byte("test content 1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		testFilePath2 := filepath.Join(srcDir2, "testfile2.txt")
		err = os.WriteFile(testFilePath2, []byte("test content 2"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		configs := Configurations{
			Configs: []Configuration{
				{
					Name:          "Config1",
					Directories:   []string{srcDir1},
					Destination:   dstDir1,
					CheckInterval: "5s",
					MinFreeSpace:  100 * 1024 * 1024, // 100 MB
				},
				{
					Name:          "Config2",
					Directories:   []string{srcDir2},
					Destination:   dstDir2,
					CheckInterval: "5s",
					MinFreeSpace:  100 * 1024 * 1024, // 100 MB
				},
			},
		}

		db := setupTestDB(t)
		defer db.Close()

		for _, config := range configs.Configs {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go MonitorAndMirror(ctx, db, config.Directories, config.Destination, config.CheckInterval, config.MinFreeSpace)

			// Allow some time for the monitor to run
			time.Sleep(15 * time.Second)

			// Check if the file was copied
			for _ = range config.Directories {
				for _, file := range []string{"testfile1.txt", "testfile2.txt"} {

					copiedFilePath := filepath.Join(config.Destination, file)
					if _, err := os.Stat(copiedFilePath); os.IsNotExist(err) {
						if "testfile2.txt" == file && config.Destination == dstDir1 {
							continue
						} else if "testfile1.txt" == file && config.Destination == dstDir2 {
							continue
						} else {
							t.Fatalf("File was not copied: %v", copiedFilePath)
						}
					}

					// Verify the content of the copied file
					copiedContent, err := os.ReadFile(copiedFilePath)
					if err != nil {
						if "testfile2.txt" == file && config.Destination == dstDir1 {
							continue
						} else if "testfile1.txt" == file && config.Destination == dstDir2 {
							continue
						} else {
							t.Fatalf("Failed to read copied file: %v", err)
						}
					}
					expectedContent := "test content 1"
					if file == "testfile2.txt" {
						expectedContent = "test content 2"
					}
					if string(copiedContent) != expectedContent {
						t.Fatalf("Copied file content mismatch: got %v, want %v", string(copiedContent), expectedContent)
					}
				}
			}
		}
	}
*/
func TestMonitorAndMirror(t *testing.T) {
	// Define fixed paths for source and destination directories
	srcDir1 := "test_src1"
	dstDir1 := "test_dst1"
	srcDir2 := "test_src2"
	dstDir2 := "test_dst2"

	// Create source and destination directories
	if err := os.MkdirAll(srcDir1, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir1)

	if err := os.MkdirAll(dstDir1, 0755); err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(dstDir1)

	if err := os.MkdirAll(srcDir2, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir2)

	if err := os.MkdirAll(dstDir2, 0755); err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(dstDir2)

	// Create test files in the source directories
	testFilePath1 := filepath.Join(srcDir1, "testfile1.txt")
	err := os.WriteFile(testFilePath1, []byte("test content 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFilePath2 := filepath.Join(srcDir2, "testfile2.txt")
	err = os.WriteFile(testFilePath2, []byte("test content 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	configs := Configurations{
		Configs: []Configuration{
			{
				Name:          "Config1",
				Directories:   []string{srcDir1},
				Destination:   dstDir1,
				CheckInterval: "3s",
				MinFreeSpace:  100 * 1024 * 1024, // 100 MB
			},
			{
				Name:          "Config2",
				Directories:   []string{srcDir2},
				Destination:   dstDir2,
				CheckInterval: "3s",
				MinFreeSpace:  100 * 1024 * 1024, // 100 MB
			},
		},
		SlackToken:     "",
		SlackChannelID: "",
	}

	db := setupTestDB(t)
	defer db.Close()

	var wg sync.WaitGroup

	for _, config := range configs.Configs {
		wg.Add(1)
		go func(config Configuration) {
			defer wg.Done()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go MonitorAndMirror(ctx, db, configs)
			// Allow some time for the monitor to run
			time.Sleep(10 * time.Second)

			// Check if the file was copied
			for _ = range config.Directories {
				for _, file := range []string{"testfile1.txt", "testfile2.txt"} {
					copiedFilePath := filepath.Join(config.Destination, file)
					if _, err := os.Stat(copiedFilePath); os.IsNotExist(err) {
						if "testfile2.txt" == file && config.Destination == dstDir1 {
							continue
						} else if "testfile1.txt" == file && config.Destination == dstDir2 {
							continue
						} else {
							t.Fatalf("File was not copied: %v", copiedFilePath)
						}
					}

					// Verify the content of the copied file
					copiedContent, err := os.ReadFile(copiedFilePath)
					if err != nil {
						if "testfile2.txt" == file && config.Destination == dstDir1 {
							continue
						} else if "testfile1.txt" == file && config.Destination == dstDir2 {
							continue
						} else {
							t.Fatalf("Failed to read copied file: %v", err)
						}
					}
					expectedContent := "test content 1"
					if file == "testfile2.txt" {
						expectedContent = "test content 2"
					}
					if string(copiedContent) != expectedContent {
						t.Fatalf("Copied file content mismatch: got %v, want %v", string(copiedContent), expectedContent)
					}
				}
			}
		}(config)
	}

	wg.Wait()
}
