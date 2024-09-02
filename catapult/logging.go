// catapult/logging.go
package catapult

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	LogFile        *os.File
	logChannel           = make(chan string, 100)
	MaxLogFileSize int64 = 10 * 1024 * 1024 // 10 MB
	wg             sync.WaitGroup
)

// StartLogger initializes the logger and starts the log writer goroutine.
func StartLogger(logFilePath string) error {
	var err error
	LogFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening log file: %v", err)
	}

	wg.Add(1)
	go logWriter()
	return nil
}

// logWriter listens to the log channel and writes log messages to the log file.
func logWriter() {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case logMessage, ok := <-logChannel:
			if !ok {
				// Channel is closed, flush remaining messages
				return
			}
			LogFile.WriteString(logMessage)
		case <-ticker.C:
			rotateLogFileIfNeeded()
		}
	}
}

// LogWithDatetime logs a message with the current datetime.
// If logToFile is true, the message is also written to the log file.
func LogWithDatetime(message string, logToFile bool) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("%s: %s\n", timestamp, message)
	fmt.Print(logMessage)
	if logToFile {
		logChannel <- logMessage
	}
}

// rotateLogFileIfNeeded checks the log file size and rotates it if it exceeds the maximum size.
func rotateLogFileIfNeeded() {
	fileInfo, err := LogFile.Stat()
	if err != nil {
		fmt.Println("Error getting log file info:", err)
		return
	}

	if fileInfo.Size() >= MaxLogFileSize {
		logFileName := LogFile.Name()
		LogFile.Close()
		timestamp := time.Now().Format("20060102-150405")
		newLogFileName := fmt.Sprintf("transfer-%s.log", timestamp)
		os.Rename(logFileName, newLogFileName)
		LogFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening new log file:", err)
		}
	}
}

// CloseLogger closes the log channel and waits for the log writer to finish.
func CloseLogger() {
	close(logChannel)
	wg.Wait()
	LogFile.Close()
}
