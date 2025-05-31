package llm

import (
	"fmt"

	"auto-api-tester/internal/logger"
)

// NewClient creates a new LLM client based on the provider
func NewClient(config *Config, logger *logger.Logger) (LLMClient, error) {
	switch config.Provider {
	case "openai":
		return NewOpenAIClient(config, logger), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
}
