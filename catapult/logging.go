// catapult/logging.go
package catapult

import (
	"fmt"
	"os"
	"time"
)

var LogFile *os.File

func LogWithDatetime(v ...interface{}) {
	message := fmt.Sprintln(append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, v...)...)
	fmt.Print(message)
	if LogFile != nil {
		LogFile.WriteString(message)
	}
}
