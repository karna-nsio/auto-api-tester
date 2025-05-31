package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"auto-api-tester/internal/logger"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient implements the LLMClient interface using OpenAI's API
type OpenAIClient struct {
	*BaseClient
	client *openai.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config *Config, logger *logger.Logger) *OpenAIClient {
	client := openai.NewClient(config.APIKey)
	return &OpenAIClient{
		BaseClient: NewBaseClient(config, logger),
		client:     client,
	}
}

// callLLM implements the actual LLM API call for OpenAI
func (c *OpenAIClient) callLLM(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       c.config.Model,
			Temperature: float32(c.config.Temperature),
			MaxTokens:   c.config.MaxTokens,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant that analyzes data and generates test data. Always respond in the requested format.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// ValidateResponse validates the LLM response format
func (c *OpenAIClient) ValidateResponse(response string, expectedType interface{}) error {
	if err := json.Unmarshal([]byte(response), expectedType); err != nil {
		return fmt.Errorf("invalid response format: %w", err)
	}
	return nil
}
