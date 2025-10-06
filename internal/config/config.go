package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

// Config represents the JSON file structure
type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

// Read reads the JSON file found at ~/.gatorconfig.json and returns a Config struct
func Read() (*Config, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config file path: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// SetUser sets the current_user_name field and writes the config to the JSON file
func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	return write(*c)
}

// getConfigFilePath returns the full path to the config file in the home directory
func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configFileName), nil
}

// write writes the config struct to the JSON file
func write(cfg Config) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("failed to get config file path: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}