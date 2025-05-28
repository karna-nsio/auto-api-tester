package config

import (
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	Environment Environment
	Test        TestConfig
	Reporting   ReportingConfig
}

// Environment holds environment-specific configuration
type Environment struct {
	BaseURL string
	Auth    AuthConfig
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Type  string
	Token string
}

// TestConfig holds test execution configuration
type TestConfig struct {
	Concurrent bool
	MaxWorkers int
	Timeout    int
	Retry      RetryConfig
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	Attempts int
	Delay    int
}

// ReportingConfig holds reporting configuration
type ReportingConfig struct {
	Format    []string
	OutputDir string
	Detailed  bool
}

// LoadConfig loads the configuration from environment variables and config files
func LoadConfig() (*Config, error) {
	// TODO: Implement configuration loading logic
	return &Config{
		Environment: Environment{
			BaseURL: "",
			Auth: AuthConfig{
				Type:  "bearer",
				Token: os.Getenv("AUTH_TOKEN"),
			},
		},
		Test: TestConfig{
			Concurrent: true,
			MaxWorkers: 5,
			Timeout:    30,
			Retry: RetryConfig{
				Attempts: 3,
				Delay:    1,
			},
		},
		Reporting: ReportingConfig{
			Format:    []string{"html", "json"},
			OutputDir: filepath.Join("reports"),
			Detailed:  true,
		},
	}, nil
}
