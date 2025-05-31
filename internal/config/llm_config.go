package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// LLMConfig holds configuration for LLM services
type LLMConfig struct {
	Provider string `json:"provider"` // e.g., "openai"
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`    // e.g., "gpt-4"
	BaseURL  string `json:"base_url"` // Optional, for custom endpoints
}

// LoadLLMConfig loads LLM configuration from a file
func LoadLLMConfig(path string) (*LLMConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read LLM config file: %v", err)
	}

	var config LLMConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse LLM config: %v", err)
	}

	// Validate required fields
	if config.Provider == "" {
		return nil, fmt.Errorf("LLM provider is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if config.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	return &config, nil
}

// SaveLLMConfig saves LLM configuration to a file
func SaveLLMConfig(config *LLMConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal LLM config: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write LLM config file: %v", err)
	}

	return nil
}
