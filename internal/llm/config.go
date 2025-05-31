package llm

// Config represents the configuration for LLM integration
type Config struct {
	// Provider specifies which LLM provider to use (e.g., "openai", "anthropic")
	Provider string `json:"provider"`

	// APIKey is the API key for the LLM provider
	APIKey string `json:"api_key"`

	// Model specifies which model to use (e.g., "gpt-4", "claude-2")
	Model string `json:"model"`

	// Temperature controls the randomness of the output (0.0 to 1.0)
	Temperature float64 `json:"temperature"`

	// MaxTokens limits the length of the generated response
	MaxTokens int `json:"max_tokens"`

	// AnalysisConfig contains specific configuration for analysis tasks
	AnalysisConfig struct {
		// SampleSize is the number of rows to analyze for patterns
		SampleSize int `json:"sample_size"`

		// MinConfidence is the minimum confidence threshold for pattern detection
		MinConfidence float64 `json:"min_confidence"`

		// EnableBusinessRules enables business rule analysis
		EnableBusinessRules bool `json:"enable_business_rules"`

		// EnableRelationshipAnalysis enables relationship analysis
		EnableRelationshipAnalysis bool `json:"enable_relationship_analysis"`
	} `json:"analysis_config"`
}

// NewDefaultConfig returns a default configuration
func NewDefaultConfig() *Config {
	return &Config{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   2000,
		AnalysisConfig: struct {
			SampleSize                 int     `json:"sample_size"`
			MinConfidence              float64 `json:"min_confidence"`
			EnableBusinessRules        bool    `json:"enable_business_rules"`
			EnableRelationshipAnalysis bool    `json:"enable_relationship_analysis"`
		}{
			SampleSize:                 100,
			MinConfidence:              0.8,
			EnableBusinessRules:        true,
			EnableRelationshipAnalysis: true,
		},
	}
}
