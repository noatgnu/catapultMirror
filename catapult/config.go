// catapult/config.go
package catapult

import (
	"encoding/json"
	"os"
)

type Configuration struct {
	Name          string   `json:"name"`
	Directories   []string `json:"directories"`
	Destination   string   `json:"destination"`
	CheckInterval string   `json:"check_interval"`
	MinFreeSpace  int64    `json:"min_free_space"`
	MinFileSize   int64    `json:"min_file_size"` // New field for minimum file size
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
		Directories:   []string{"exampleDir1", "exampleDir2"},
		Destination:   "exampleDestinationDir",
		CheckInterval: "1m",
		MinFreeSpace:  10000 * 1024 * 1024, // 10 GB
		MinFileSize:   1024 * 1024,         // 1 MB
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
