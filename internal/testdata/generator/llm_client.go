package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// LLMClient handles interactions with the LLM service
type LLMClient struct {
	client  *openai.Client
	model   string
	baseURL string
}

// NewLLMClient creates a new LLM client
func NewLLMClient(apiKey, model, baseURL string) *LLMClient {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	return &LLMClient{
		client:  openai.NewClientWithConfig(config),
		model:   model,
		baseURL: baseURL,
	}
}

// GenerateMappingSuggestion generates a mapping suggestion using LLM
func (c *LLMClient) GenerateMappingSuggestion(ctx context.Context, tableInfo TableInfo) (*SchemaMapping, error) {
	prompt := fmt.Sprintf(`Analyze the following database table and suggest a mapping to an API entity:

Table Name: %s
Columns:
%s

Please suggest:
1. A suitable API entity name
2. Field mappings between table columns and API fields
3. Any potential business rules or relationships

Respond in JSON format with the following structure:
{
    "api_entity_name": "string",
    "field_mappings": {"column_name": "api_field_name"},
    "business_rules": [{"type": "string", "condition": "string", "action": "string", "priority": 1}],
    "relationships": [{"type": "string", "source_entity": "string", "target_entity": "string", "source_field": "string", "target_field": "string"}]
}`, tableInfo.Name, formatTableInfo(tableInfo))

	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mapping suggestion: %v", err)
	}

	var mapping SchemaMapping
	if err := json.Unmarshal([]byte(response), &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse mapping suggestion: %v", err)
	}

	return &mapping, nil
}

// ExtractBusinessRules extracts business rules using LLM
func (c *LLMClient) ExtractBusinessRules(ctx context.Context, data []interface{}) ([]BusinessRule, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %v", err)
	}

	prompt := fmt.Sprintf(`Analyze the following data and extract business rules:

%s

Please identify any patterns, constraints, or business rules that should be enforced.
Respond in JSON format with an array of business rules:
[
    {
        "type": "validation|transformation|dependency",
        "condition": "rule condition",
        "action": "action to take",
        "priority": 1
    }
]`, string(dataJSON))

	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract business rules: %v", err)
	}

	var rules []BusinessRule
	if err := json.Unmarshal([]byte(response), &rules); err != nil {
		return nil, fmt.Errorf("failed to parse business rules: %v", err)
	}

	return rules, nil
}

// ValidateData validates data using LLM
func (c *LLMClient) ValidateData(ctx context.Context, data interface{}, rules []BusinessRule) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %v", err)
	}

	prompt := fmt.Sprintf(`Validate the following data against the given business rules:

Data:
%s

Rules:
%s

Please check if the data complies with all rules. Respond with a JSON object:
{
    "valid": true|false,
    "violations": [
        {
            "rule": "rule description",
            "message": "violation message"
        }
    ]
}`, string(dataJSON), string(rulesJSON))

	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to validate data: %v", err)
	}

	var result struct {
		Valid      bool `json:"valid"`
		Violations []struct {
			Rule    string `json:"rule"`
			Message string `json:"message"`
		} `json:"violations"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("failed to parse validation result: %v", err)
	}

	if !result.Valid {
		var messages []string
		for _, v := range result.Violations {
			messages = append(messages, fmt.Sprintf("%s: %s", v.Rule, v.Message))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
	}

	return nil
}

// TransformData transforms data according to business rules
func (c *LLMClient) TransformData(ctx context.Context, data interface{}, rules []BusinessRule) (interface{}, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %v", err)
	}

	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rules: %v", err)
	}

	prompt := fmt.Sprintf(`Transform the following data according to the given business rules:

Data:
%s

Rules:
%s

Please transform the data to comply with all rules. Respond with the transformed data in the same format as the input.`, string(dataJSON), string(rulesJSON))

	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to transform data: %v", err)
	}

	var transformedData interface{}
	if err := json.Unmarshal([]byte(response), &transformedData); err != nil {
		return nil, fmt.Errorf("failed to parse transformed data: %v", err)
	}

	return transformedData, nil
}

// AnalyzeRelationships analyzes relationships between entities using LLM
func (c *LLMClient) AnalyzeRelationships(ctx context.Context, tables map[string]TableInfo) ([]Relationship, error) {
	tablesJSON, err := json.Marshal(tables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tables: %v", err)
	}

	prompt := fmt.Sprintf(`Analyze the following database tables and identify relationships:

%s

Please identify all relationships between tables, including:
1. Foreign key relationships
2. Many-to-many relationships
3. Inheritance relationships
4. Custom relationships

Respond in JSON format with an array of relationships:
[
    {
        "type": "foreign_key|many_to_many|inheritance|custom",
        "source_entity": "table_name",
        "target_entity": "table_name",
        "source_field": "column_name",
        "target_field": "column_name"
    }
]`, string(tablesJSON))

	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze relationships: %v", err)
	}

	var relationships []Relationship
	if err := json.Unmarshal([]byte(response), &relationships); err != nil {
		return nil, fmt.Errorf("failed to parse relationships: %v", err)
	}

	return relationships, nil
}

// SuggestFieldMappings suggests field mappings using LLM
func (c *LLMClient) SuggestFieldMappings(ctx context.Context, tableColumns []ColumnInfo, apiFields []string) (map[string]string, error) {
	columnsJSON, err := json.Marshal(tableColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal columns: %v", err)
	}

	fieldsJSON, err := json.Marshal(apiFields)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields: %v", err)
	}

	prompt := fmt.Sprintf(`Suggest mappings between database columns and API fields:

Columns:
%s

API Fields:
%s

Please suggest appropriate mappings between columns and fields.
Respond in JSON format with a map of column names to field names:
{
    "column_name": "field_name"
}`, string(columnsJSON), string(fieldsJSON))

	response, err := c.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest field mappings: %v", err)
	}

	var mappings map[string]string
	if err := json.Unmarshal([]byte(response), &mappings); err != nil {
		return nil, fmt.Errorf("failed to parse field mappings: %v", err)
	}

	return mappings, nil
}

// callLLM makes a call to the LLM service
func (c *LLMClient) callLLM(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant that analyzes database schemas and suggests mappings to API entities. Always respond in valid JSON format.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.7,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

// formatTableInfo formats table information for the prompt
func formatTableInfo(table TableInfo) string {
	var sb strings.Builder
	for _, col := range table.Columns {
		sb.WriteString(fmt.Sprintf("- %s (%s)", col.Name, col.Type))
		if col.IsPrimary {
			sb.WriteString(" [Primary Key]")
		}
		if col.IsForeign {
			sb.WriteString(fmt.Sprintf(" [Foreign Key -> %s]", col.References))
		}
		if col.Nullable {
			sb.WriteString(" [Nullable]")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
