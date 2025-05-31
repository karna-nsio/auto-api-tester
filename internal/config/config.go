package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"auto-api-tester/internal/llm"
)

// Config represents the application configuration
type Config struct {
	Test struct {
		Concurrent bool `json:"concurrent"`
		MaxWorkers int  `json:"max_workers"`
		Timeout    int  `json:"timeout"`
		Retry      struct {
			Attempts int `json:"attempts"`
			Delay    int `json:"delay"`
		} `json:"retry"`
	} `json:"test"`

	Reporting struct {
		Format    string `json:"format"`
		OutputDir string `json:"output_dir"`
		Detailed  bool   `json:"detailed"`
	} `json:"reporting"`

	LLM *llm.Config `json:"llm,omitempty"`
}

// LoadConfig loads the configuration from a file
func LoadConfig() (*Config, error) {
	// Default config path
	configPath := "config/config.json"

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		config := &Config{
			Test: struct {
				Concurrent bool `json:"concurrent"`
				MaxWorkers int  `json:"max_workers"`
				Timeout    int  `json:"timeout"`
				Retry      struct {
					Attempts int `json:"attempts"`
					Delay    int `json:"delay"`
				} `json:"retry"`
			}{
				Concurrent: true,
				MaxWorkers: 5,
				Timeout:    30,
				Retry: struct {
					Attempts int `json:"attempts"`
					Delay    int `json:"delay"`
				}{
					Attempts: 3,
					Delay:    5,
				},
			},
			Reporting: struct {
				Format    string `json:"format"`
				OutputDir string `json:"output_dir"`
				Detailed  bool   `json:"detailed"`
			}{
				Format:    "json",
				OutputDir: "reports",
				Detailed:  true,
			},
			LLM: llm.NewDefaultConfig(),
		}

		// Create config directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %v", err)
		}

		// Write default config
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default config: %v", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write default config: %v", err)
		}

		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse config
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Set default LLM config if not provided
	if config.LLM == nil {
		config.LLM = llm.NewDefaultConfig()
	}

	return &config, nil
}
