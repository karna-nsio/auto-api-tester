package llm

import (
	"context"
)

// AnalysisResult represents the result of LLM analysis
type AnalysisResult struct {
	// DataPatterns contains analyzed patterns for a column
	DataPatterns struct {
		DataType    string        `json:"dataType"`
		Format      string        `json:"format"`
		ValueRange  []interface{} `json:"valueRange"`
		Patterns    []string      `json:"patterns"`
		Constraints []string      `json:"constraints"`
	} `json:"dataPatterns"`

	// BusinessRules contains inferred business rules
	BusinessRules struct {
		Rules       []string      `json:"rules"`
		Constraints []string      `json:"constraints"`
		TestData    []interface{} `json:"testData"`
	} `json:"businessRules"`
}

// Relationship represents a foreign key relationship
type Relationship struct {
	Table            string `json:"table"`
	Column           string `json:"column"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
}

// LLMClient defines the interface for LLM interactions
type LLMClient interface {
	// AnalyzeColumn analyzes a column's data patterns
	AnalyzeColumn(ctx context.Context, tableName, columnName string, sampleData []interface{}) (*AnalysisResult, error)

	// AnalyzeRelationships analyzes table relationships
	AnalyzeRelationships(ctx context.Context, tableName string, schema map[string]interface{}) (*EnhancedAnalysisResult, error)

	// AnalyzeBusinessRules analyzes business rules from existing data
	AnalyzeBusinessRules(ctx context.Context, tableName string, sampleData []map[string]interface{}) (interface{}, error)

	// ValidateTestData validates generated test data against business rules
	ValidateTestData(ctx context.Context, tableName string, testData map[string]interface{}, rules *AnalysisResult) (bool, error)

	// GenerateTestData generates test data based on analysis
	GenerateTestData(ctx context.Context, tableName string, analysis *AnalysisResult) (map[string]interface{}, error)

	// callLLM handles the actual LLM API call
	callLLM(ctx context.Context, prompt string) (string, error)
}
