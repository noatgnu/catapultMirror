// catapult/logging.go
package catapult

import (
	"fmt"
	"os"
	"time"
)

var LogFile *os.File

func LogWithDatetime(message string, logToFile bool) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("%s: %s\n", timestamp, message)
	fmt.Print(logMessage)
	if logToFile && LogFile != nil {
		LogFile.WriteString(logMessage)
	}
}
