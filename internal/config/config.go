package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Environment Environment
	Test        TestConfig
	Reporting   ReportingConfig
}

// Environment holds environment-specific configuration
type Environment struct {
	BaseURL string `yaml:"base_url"`
	Auth    AuthConfig
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Type  string `yaml:"type"`
	Token string `yaml:"token"`
}

// TestConfig holds test execution configuration
type TestConfig struct {
	Concurrent bool        `yaml:"concurrent"`
	MaxWorkers int         `yaml:"max_workers"`
	Timeout    int         `yaml:"timeout"`
	Retry      RetryConfig `yaml:"retry"`
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	Attempts int `yaml:"attempts"`
	Delay    int `yaml:"delay"`
}

// ReportingConfig holds reporting configuration
type ReportingConfig struct {
	Format    []string `yaml:"format"`
	OutputDir string   `yaml:"output_dir"`
	Detailed  bool     `yaml:"detailed"`
}

// LoadConfig loads the configuration from environment variables and config files
func LoadConfig() (*Config, error) {
	// Default config file path
	configPath := "config/config.yaml"

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s", configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Override auth token from environment variable if set
	if token := os.Getenv("AUTH_TOKEN"); token != "" {
		config.Environment.Auth.Token = token
	}

	// Set default values if not specified
	if config.Test.MaxWorkers == 0 {
		config.Test.MaxWorkers = 5
	}
	if config.Test.Timeout == 0 {
		config.Test.Timeout = 30
	}
	if config.Test.Retry.Attempts == 0 {
		config.Test.Retry.Attempts = 3
	}
	if config.Test.Retry.Delay == 0 {
		config.Test.Retry.Delay = 1
	}
	if len(config.Reporting.Format) == 0 {
		config.Reporting.Format = []string{"json"}
	}
	if config.Reporting.OutputDir == "" {
		config.Reporting.OutputDir = filepath.Join("reports")
	}

	return &config, nil
}
