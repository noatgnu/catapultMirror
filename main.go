// main.go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/noatgnu/catapultMirror/catapult"
)

func main() {
	configFile := flag.String("config", "", "Path to the JSON configuration file")
	dbPath := flag.String("db", "file_sizes.db", "Path to the SQLite database file")
	logFilePath := flag.String("log", "transfer.log", "Path to the log file")
	flag.Parse()

	var err error
	catapult.LogFile, err = os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer catapult.LogFile.Close()

	if *configFile == "" {
		catapult.LogWithDatetime("Usage: catapultMirror -config=<config_file> -db=<db_file> -log=<log_file>")
		return
	}

	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		err := catapult.CreateTemplateConfig(*configFile)
		if err != nil {
			catapult.LogWithDatetime("Error creating template configuration file:", err)
			return
		}
		catapult.LogWithDatetime(fmt.Sprintf("Template configuration file created at %s. Please fill in the file and start again.", *configFile))
		return
	}

	config, err := catapult.ReadConfigFromFile(*configFile)
	if err != nil {
		catapult.LogWithDatetime("Error reading configuration file:", err)
		return
	}

	db, err := catapult.InitDB(*dbPath)
	if err != nil {
		catapult.LogWithDatetime("Error initializing database:", err)
		return
	}
	defer db.Close()

	freeSpace, err := catapult.GetFreeSpace(config.Destination)
	if err != nil {
		catapult.LogWithDatetime("Error getting free space:", err)
		return
	}

	catapult.LogWithDatetime(fmt.Sprintf("Destination free space: %.2f MB", float64(freeSpace)/1024/1024))

	catapult.MonitorAndMirror(db, config.Directories, config.Destination, config.CheckInterval, config.MinFreeSpace)
}
