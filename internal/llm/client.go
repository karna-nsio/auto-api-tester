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

// TableAnalysisRequest represents the input for table analysis
type TableAnalysisRequest struct {
	TableName  string                 `json:"tableName"`
	APIContext APIContext             `json:"apiContext"`
	Schema     map[string]interface{} `json:"schema"`
}

// APIContext contains information about the API endpoint
type APIContext struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// TableSuggestion represents a suggested table with reasoning
type TableSuggestion struct {
	TableName       string  `json:"tableName"`
	SimilarityScore float64 `json:"similarityScore"`
	Reasoning       string  `json:"reasoning"`
}

// AnalysisResult is defined in types.go

// EnhancedAnalysisResult extends the original AnalysisResult
type EnhancedAnalysisResult struct {
	Relationships *AnalysisResult   `json:"relationships"`
	Suggestions   []TableSuggestion `json:"suggestions"`
	SimilarTables []struct {
		Table1    string `json:"table1"`
		Table2    string `json:"table2"`
		Reasoning string `json:"reasoning"`
	} `json:"similarTables"`
	ForeignKeysAndDependencies []struct {
		Table      string `json:"table"`
		ForeignKey string `json:"foreignKey"`
		References struct {
			Table  string `json:"table"`
			Column string `json:"column"`
		} `json:"references"`
	} `json:"foreignKeysAndDependencies"`
}

// AnalyzeRelationships implements the LLMClient interface
func (c *BaseClient) AnalyzeRelationships(ctx context.Context, tableName string, schema map[string]interface{}) (*EnhancedAnalysisResult, error) {
	// Prepare the prompt for relationship analysis - optimized for token usage
	schemaJSON, _ := json.Marshal(schema) // Remove indentation to save tokens

	prompt := fmt.Sprintf(`Analyze schema relationships for table "%s":
Schema: %s

Find:
1. Foreign keys and dependencies
2. Similar tables with reasoning
3. Key relationships

Respond in JSON matching EnhancedAnalysisResult structure.`,
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

	// Format and display the response
	formattedResponse, err := json.MarshalIndent(json.RawMessage(response), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}
	fmt.Printf("LLM Analysis Response:\n%s\n", string(formattedResponse))

	// Parse the response into EnhancedAnalysisResult
	var enhancedResult EnhancedAnalysisResult
	if err := json.Unmarshal([]byte(response), &enhancedResult); err != nil {
		c.logger.LogLLMInteraction("AnalyzeRelationships", map[string]interface{}{
			"table":  tableName,
			"schema": schema,
		}, nil, err)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.logger.LogLLMInteraction("AnalyzeRelationships", map[string]interface{}{
		"table":  tableName,
		"schema": schema,
	}, enhancedResult, nil)

	return &enhancedResult, nil
}

// AnalyzeBusinessRules implements the LLMClient interface
func (c *BaseClient) AnalyzeBusinessRules(ctx context.Context, tableName string, sampleData []map[string]interface{}) (interface{}, error) {
	// Extract context from sample data
	context := sampleData[0]
	endpoint := context["endpoint"].(map[string]interface{})
	sampleRecord := context["sampleRecord"].(map[string]interface{})

	// Prepare the prompt for business rules analysis
	sampleJSON, _ := json.MarshalIndent(sampleRecord, "", "  ")
	templateJSON, _ := json.MarshalIndent(endpoint["body"], "", "  ")

	// Create a dynamic example structure based on the template
	var exampleStructure interface{}
	if err := json.Unmarshal(templateJSON, &exampleStructure); err == nil {
		// If template is an array, use its first element as example
		if arr, ok := exampleStructure.([]interface{}); ok && len(arr) > 0 {
			exampleStructure = arr[0]
		}
	}
	exampleJSON, _ := json.MarshalIndent(exampleStructure, "", "  ")

	prompt := fmt.Sprintf(`You are an intelligent test data generator. Based on the following API specification and sample database record, generate a fully populated test data object for the %s endpoint:

**Endpoint**: %s %s

### 1. API Request Body Template:
%s

### 2. Sample Database Record:
%s

### Your Task:
1. Analyze the API template and the sample database record.
2. Identify valid data types, formats, and constraints.
3. Generate a realistic test data object (with sample values) that matches the structure of the API request body.
4. Ensure generated data follows business logic and inferred validation rules (e.g., valid email, proper phone format, realistic DOB).
5. If the request template fields use different names than the database (e.g., 'is_activated' vs 'is_active'), map accordingly.

### Output Format:
Respond with a single JSON object that matches the structure of the API request body template.

Example structure (based on your API template):
%s`,
		endpoint["method"], endpoint["method"], endpoint["path"],
		string(templateJSON),
		string(sampleJSON),
		string(exampleJSON))

	// Call LLM and parse response
	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		c.logger.LogLLMInteraction("AnalyzeBusinessRules", map[string]interface{}{
			"table":    tableName,
			"endpoint": endpoint,
			"sample":   sampleRecord,
		}, nil, err)
		return nil, fmt.Errorf("failed to analyze business rules: %w", err)
	}

	fmt.Println("prompt: ", prompt)
	fmt.Println("llm response: ", response)

	// Parse the response into a single object first
	var testDataObj interface{}
	if err := json.Unmarshal([]byte(response), &testDataObj); err != nil {
		c.logger.LogLLMInteraction("AnalyzeBusinessRules", map[string]interface{}{
			"table":    tableName,
			"endpoint": endpoint,
			"sample":   sampleRecord,
		}, nil, err)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.logger.LogLLMInteraction("AnalyzeBusinessRules", map[string]interface{}{
		"table":    tableName,
		"endpoint": endpoint,
		"sample":   sampleRecord,
	}, testDataObj, nil)

	return testDataObj, nil
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

// callLLM handles the LLM API call based on the configured provider
func (c *BaseClient) callLLM(ctx context.Context, prompt string) (string, error) {
	// Create a new client based on the provider
	client, err := NewClient(c.config, c.logger)
	if err != nil {
		return "", fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Call the specific client's implementation directly
	return client.callLLM(ctx, prompt)
}
