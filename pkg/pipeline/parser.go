package pipeline

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseConfig reads the .rune.yaml file from a given directory path
// and unmarshals it into a Config struct.
func ParseConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("pipeline configuration file not found at %s", filepath)
		}
		return nil, fmt.Errorf("failed to read pipeline configuration file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	return &config, nil
}
