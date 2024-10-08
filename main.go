// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/noatgnu/catapultMirror/catapult"
)

func main() {
	configFile := flag.String("config", "", "Path to the JSON configuration file")
	dbPath := flag.String("db", "file_sizes.db", "Path to the SQLite database file")
	logFilePath := flag.String("log", "transfer.log", "Path to the log file")
	flag.Parse()

	err := catapult.StartLogger(*logFilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer catapult.CloseLogger()

	if *configFile == "" {
		catapult.LogWithDatetime("Usage: catapultMirror -config=<config_file> -db=<db_file> -log=<log_file>", false)
		return
	}

	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		err := catapult.CreateTemplateConfig(*configFile)
		if err != nil {
			catapult.LogWithDatetime(fmt.Sprintf("Error creating template configuration file: %v", err), false)
			return
		}
		catapult.LogWithDatetime(fmt.Sprintf("Template configuration file created at %s. Please fill in the file and start again.", *configFile), false)
		return
	}

	configs, err := catapult.ReadConfigsFromFile(*configFile)
	if err != nil {
		catapult.LogWithDatetime(fmt.Sprintf("Error reading configuration file: %v", err), false)
		return
	}

	db, err := catapult.InitDB(*dbPath)
	if err != nil {
		catapult.LogWithDatetime(fmt.Sprintf("Error initializing database: %v", err), false)
		return
	}
	defer db.Close()

	// Initialize Slack with the configurations
	catapult.InitSlack(configs)

	var wg sync.WaitGroup

	for _, config := range configs.Configs {
		wg.Add(1)
		go func(config catapult.Configuration) {
			defer wg.Done()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			catapult.MonitorAndMirror(ctx, db, configs)
		}(config)
	}

	wg.Wait()
}
