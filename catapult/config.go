// catapult/config.go
package catapult

import (
	"encoding/json"
	"os"
	"time"
)

type Configuration struct {
	Name                string   `json:"name"`
	Directories         []string `json:"directories"`
	Destinations        []string `json:"destinations"`
	CheckInterval       string   `json:"check_interval"`
	MinFreeSpace        int64    `json:"min_free_space"`
	MinFileSize         int64    `json:"min_file_size"`
	OverrideIfDifferent bool     `json:"override_if_different"`
}
type Configurations struct {
	Configs        []Configuration `json:"configs"`
	SlackToken     string          `json:"slack_token,omitempty"`
	SlackChannelID string          `json:"slack_channel_id,omitempty"`
}

// CreateTemplateConfig creates a template configuration file with example values.
//
// Parameters:
// - filePath: The path where the template configuration file will be created.
//
// Returns:
// - error: An error object if there was an issue creating the file.
func CreateTemplateConfig(filePath string) error {
	templateConfig := Configuration{
		Directories:         []string{"exampleDir1", "exampleDir2"},
		Destinations:        []string{"exampleDestinationDir", "exampleDestinationDir2"},
		CheckInterval:       "1m",
		MinFreeSpace:        10000 * 1024 * 1024, // 10 GB
		MinFileSize:         1024 * 1024,         // 1 MB
		OverrideIfDifferent: false,
	}

	templateConfigs := Configurations{
		Configs:        []Configuration{templateConfig},
		SlackToken:     "",
		SlackChannelID: "",
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(templateConfigs)
}

// ReadConfigFromFile reads a single configuration from a file.
//
// Parameters:
// - filePath: The path of the configuration file to read.
//
// Returns:
// - Configuration: The configuration read from the file.
// - error: An error object if there was an issue reading the file.
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

// ReadConfigsFromFile reads multiple configurations from a file.
//
// Parameters:
// - filePath: The path of the configuration file to read.
//
// Returns:
// - Configurations: The configurations read from the file.
// - error: An error object if there was an issue reading the file.
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
