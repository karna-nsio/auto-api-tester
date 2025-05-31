package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"auto-api-tester/internal/logger"
)

// BaseClient provides a base implementation of the LLMClient interface
type BaseClient struct {
	config *Config
	logger *logger.Logger
}

// NewBaseClient creates a new base LLM client
func NewBaseClient(config *Config, logger *logger.Logger) *BaseClient {
	return &BaseClient{
		config: config,
		logger: logger,
	}
}

// AnalyzeColumn implements the LLMClient interface
func (c *BaseClient) AnalyzeColumn(ctx context.Context, tableName, columnName string, sampleData []interface{}) (*AnalysisResult, error) {
	// Prepare the prompt for column analysis
	prompt := fmt.Sprintf(`Analyze the following column data from table "%s", column "%s":
Sample Data: %v

Please analyze:
1. Data type and format
2. Value ranges and patterns
3. Any constraints or special rules
4. Common patterns in the data

Respond in JSON format matching the AnalysisResult.DataPatterns structure.`,
		tableName, columnName, sampleData)

	// Call LLM and parse response
	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		c.logger.LogLLMInteraction("AnalyzeColumn", map[string]interface{}{
			"table":  tableName,
			"column": columnName,
			"data":   sampleData,
		}, nil, err)
		return nil, fmt.Errorf("failed to analyze column: %w", err)
	}

	// Parse the response into AnalysisResult
	var result AnalysisResult
	if err := json.Unmarshal([]byte(response), &result.DataPatterns); err != nil {
		c.logger.LogLLMInteraction("AnalyzeColumn", map[string]interface{}{
			"table":  tableName,
			"column": columnName,
			"data":   sampleData,
		}, nil, err)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.logger.LogLLMInteraction("AnalyzeColumn", map[string]interface{}{
		"table":  tableName,
		"column": columnName,
		"data":   sampleData,
	}, result, nil)

	return &result, nil
}

// AnalyzeRelationships implements the LLMClient interface
func (c *BaseClient) AnalyzeRelationships(ctx context.Context, tableName string, schema map[string]interface{}) (*AnalysisResult, error) {
	// Prepare the prompt for relationship analysis
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	prompt := fmt.Sprintf(`Analyze the following database schema for table "%s":
Schema: %s

Please analyze:
1. Foreign key relationships
2. Table dependencies
3. Referential integrity rules

Respond in JSON format matching the AnalysisResult.Relationships structure.`,
		tableName, string(schemaJSON))

	// Call LLM and parse response
	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		c.logger.LogLLMInteraction("AnalyzeRelationships", map[string]interface{}{
			"table":  tableName,
			"schema": schema,
		}, nil, err)
		return nil, fmt.Errorf("failed to analyze relationships: %w", err)
	}

	// Parse the response into AnalysisResult
	var result AnalysisResult
	if err := json.Unmarshal([]byte(response), &result.Relationships); err != nil {
		c.logger.LogLLMInteraction("AnalyzeRelationships", map[string]interface{}{
			"table":  tableName,
			"schema": schema,
		}, nil, err)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.logger.LogLLMInteraction("AnalyzeRelationships", map[string]interface{}{
		"table":  tableName,
		"schema": schema,
	}, result, nil)

	return &result, nil
}

// AnalyzeBusinessRules implements the LLMClient interface
func (c *BaseClient) AnalyzeBusinessRules(ctx context.Context, tableName string, sampleData []map[string]interface{}) (*AnalysisResult, error) {
	// Prepare the prompt for business rules analysis
	sampleJSON, _ := json.MarshalIndent(sampleData, "", "  ")
	prompt := fmt.Sprintf(`Analyze the following sample data from table "%s":
Sample Data: %s

Please analyze:
1. Business rules and constraints
2. Data validation rules
3. Any patterns that suggest business logic

Respond in JSON format matching the AnalysisResult.BusinessRules structure.`,
		tableName, string(sampleJSON))

	// Call LLM and parse response
	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		c.logger.LogLLMInteraction("AnalyzeBusinessRules", map[string]interface{}{
			"table": tableName,
			"data":  sampleData,
		}, nil, err)
		return nil, fmt.Errorf("failed to analyze business rules: %w", err)
	}

	// Parse the response into AnalysisResult
	var result AnalysisResult
	if err := json.Unmarshal([]byte(response), &result.BusinessRules); err != nil {
		c.logger.LogLLMInteraction("AnalyzeBusinessRules", map[string]interface{}{
			"table": tableName,
			"data":  sampleData,
		}, nil, err)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.logger.LogLLMInteraction("AnalyzeBusinessRules", map[string]interface{}{
		"table": tableName,
		"data":  sampleData,
	}, result, nil)

	return &result, nil
}

// ValidateTestData implements the LLMClient interface
func (c *BaseClient) ValidateTestData(ctx context.Context, tableName string, testData map[string]interface{}, rules *AnalysisResult) (bool, error) {
	// Prepare the prompt for validation
	testDataJSON, _ := json.MarshalIndent(testData, "", "  ")
	rulesJSON, _ := json.MarshalIndent(rules, "", "  ")
	prompt := fmt.Sprintf(`Validate the following test data for table "%s" against the business rules:
Test Data: %s
Business Rules: %s

Please validate if the test data follows all business rules and constraints.
Respond with a boolean value (true/false) and any validation errors.`,
		tableName, string(testDataJSON), string(rulesJSON))

	// Call LLM and parse response
	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		c.logger.LogLLMInteraction("ValidateTestData", map[string]interface{}{
			"table":    tableName,
			"testData": testData,
			"rules":    rules,
		}, nil, err)
		return false, fmt.Errorf("failed to validate test data: %w", err)
	}

	// Parse the boolean response
	valid := strings.TrimSpace(strings.ToLower(response)) == "true"

	c.logger.LogLLMInteraction("ValidateTestData", map[string]interface{}{
		"table":    tableName,
		"testData": testData,
		"rules":    rules,
	}, valid, nil)

	return valid, nil
}

// GenerateTestData implements the LLMClient interface
func (c *BaseClient) GenerateTestData(ctx context.Context, tableName string, analysis *AnalysisResult) (map[string]interface{}, error) {
	// Prepare the prompt for test data generation
	analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")
	prompt := fmt.Sprintf(`Generate test data for table "%s" based on the following analysis:
Analysis: %s

Please generate realistic test data that follows all patterns, relationships, and business rules.
Respond with a JSON object containing the test data.`,
		tableName, string(analysisJSON))

	// Call LLM and parse response
	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		c.logger.LogLLMInteraction("GenerateTestData", map[string]interface{}{
			"table":    tableName,
			"analysis": analysis,
		}, nil, err)
		return nil, fmt.Errorf("failed to generate test data: %w", err)
	}

	// Parse the response into a map
	var testData map[string]interface{}
	if err := json.Unmarshal([]byte(response), &testData); err != nil {
		c.logger.LogLLMInteraction("GenerateTestData", map[string]interface{}{
			"table":    tableName,
			"analysis": analysis,
		}, nil, err)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.logger.LogLLMInteraction("GenerateTestData", map[string]interface{}{
		"table":    tableName,
		"analysis": analysis,
	}, testData, nil)

	return testData, nil
}

// callLLM is a placeholder for the actual LLM API call
// This should be implemented by specific LLM providers
func (c *BaseClient) callLLM(ctx context.Context, prompt string) (string, error) {
	// This is a placeholder - actual implementation will depend on the LLM provider
	return "", fmt.Errorf("callLLM not implemented")
}
