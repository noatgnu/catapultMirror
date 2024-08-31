// catapult/config.go
package catapult

import (
	"encoding/json"
	"os"
	"time"
)

type Configuration struct {
	Name          string   `json:"name"`
	Directories   []string `json:"directories"`
	Destination   string   `json:"destination"`
	CheckInterval string   `json:"check_interval"`
	MinFreeSpace  int64    `json:"min_free_space"`
}

type Configurations struct {
	Configs        []Configuration `json:"configs"`
	SlackToken     string          `json:"slack_token,omitempty"`
	SlackChannelID string          `json:"slack_channel_id,omitempty"`
}

func CreateTemplateConfig(filePath string) error {
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

func ReadConfigFromFile(filePath string) (Configuration, error) {
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

	duration, err := time.ParseDuration(config.CheckInterval)
	if err != nil {
		return config, err
	}

	config.CheckInterval = duration.String()
	return config, nil
}

func ReadConfigsFromFile(filePath string) (Configurations, error) {
	var configs Configurations
	file, err := os.Open(filePath)
	if err != nil {
		return configs, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configs)
	return configs, err
}
