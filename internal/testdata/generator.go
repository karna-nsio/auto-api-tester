package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"auto-api-tester/internal/types"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestDataTemplate represents the structure of our test data file
type TestDataTemplate struct {
	Endpoints map[string]EndpointTestData `json:"endpoints"`
}

// EndpointTestData represents test data for a specific endpoint and method
type EndpointTestData struct {
	PathParams  map[string]interface{} `json:"path_params,omitempty"`
	QueryParams map[string]interface{} `json:"query_params,omitempty"`
	Body        interface{}            `json:"body,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
}

// Generator handles the generation of test data templates
type Generator struct {
	outputDir string
}

// NewGenerator creates a new instance of Generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
	}
}

// GenerateTemplate generates a test data template file based on endpoints
func (g *Generator) GenerateTemplate(endpoints []types.Endpoint) error {
	template := TestDataTemplate{
		Endpoints: make(map[string]EndpointTestData),
	}

	// Process each endpoint
	for _, endpoint := range endpoints {
		// Generate test data for this endpoint and method
		testData := g.generateEndpointTestData(endpoint)
		key := fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path)
		template.Endpoints[key] = testData
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Write template to file
	outputPath := filepath.Join(g.outputDir, "testdata_template.json")
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %v", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write template file: %v", err)
	}

	fmt.Printf("Test data template generated at: %s\n", outputPath)
	return nil
}

// generateEndpointTestData generates test data for a specific endpoint
func (g *Generator) generateEndpointTestData(endpoint types.Endpoint) EndpointTestData {
	testData := EndpointTestData{
		PathParams:  make(map[string]interface{}),
		QueryParams: make(map[string]interface{}),
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
	}

	// Process parameters
	for _, param := range endpoint.Parameters {
		switch param.In {
		case "path":
			testData.PathParams[param.Name] = g.generateSampleValue(param)
		case "query":
			testData.QueryParams[param.Name] = g.generateSampleValue(param)
		case "body":
			testData.Body = g.generateBodySchema(param.Schema)
		case "header":
			if value := g.generateSampleValue(param); value != nil {
				testData.Headers[param.Name] = fmt.Sprint(value)
			}
		}
	}

	return testData
}

// generateSampleValue generates a sample value based on parameter type
func (g *Generator) generateSampleValue(param types.Parameter) interface{} {
	if schema, ok := param.Schema.(map[string]interface{}); ok {
		if typeStr, ok := schema["type"].(string); ok {
			switch typeStr {
			case "string":
				if format, ok := schema["format"].(string); ok {
					switch format {
					case "email":
						return "test@example.com"
					case "date":
						return "2024-01-01"
					case "date-time":
						return "2024-01-01T12:00:00Z"
					case "uuid":
						return "123e4567-e89b-12d3-a456-426614174000"
					case "uri":
						return "https://example.com"
					case "ipv4":
						return "192.168.1.1"
					case "ipv6":
						return "2001:db8::1"
					}
				}
				if enum, ok := schema["enum"].([]interface{}); ok && len(enum) > 0 {
					return enum[0]
				}
				if pattern, ok := schema["pattern"].(string); ok {
					// Generate a simple string that matches common patterns
					switch {
					case strings.Contains(pattern, "\\d"):
						return "12345"
					case strings.Contains(pattern, "[a-zA-Z]"):
						return "abc"
					default:
						return "sample_string"
					}
				}
				return "sample_string"
			case "number":
				if format, ok := schema["format"].(string); ok {
					switch format {
					case "float":
						return 123.45
					case "double":
						return 123.456789
					}
				}
				return 123.45
			case "integer":
				if format, ok := schema["format"].(string); ok {
					switch format {
					case "int32":
						return 123
					case "int64":
						return 123456789
					}
				}
				return 123
			case "boolean":
				return true
			case "array":
				if items, ok := schema["items"].(map[string]interface{}); ok {
					itemType := "string"
					if typeStr, ok := items["type"].(string); ok {
						itemType = typeStr
					}
					switch itemType {
					case "string":
						return []string{"item1", "item2"}
					case "number":
						return []float64{1.23, 4.56}
					case "integer":
						return []int{1, 2, 3}
					case "boolean":
						return []bool{true, false}
					}
				}
				return []interface{}{"sample_item"}
			case "object":
				if properties, ok := schema["properties"].(map[string]interface{}); ok {
					result := make(map[string]interface{})
					for key, prop := range properties {
						if propMap, ok := prop.(map[string]interface{}); ok {
							result[key] = g.generateSampleValue(types.Parameter{Schema: propMap})
						}
					}
					return result
				}
				return map[string]interface{}{"key": "value"}
			}
		}
	}
	return nil
}

// generateBodySchema generates a sample body schema
func (g *Generator) generateBodySchema(schema interface{}) interface{} {
	// Handle schema reference
	if schemaRef, ok := schema.(*openapi3.SchemaRef); ok {
		if schemaRef.Ref != "" {
			// Use the referenced schema
			return g.generateBodySchema(schemaRef.Value)
		}
		schema = schemaRef.Value
	}

	if schemaMap, ok := schema.(*openapi3.Schema); ok {
		// Handle array type
		if schemaMap.Type != nil && schemaMap.Type.Is("array") {
			if schemaMap.Items != nil {
				// Generate a sample array with one item using the items schema
				itemSchema := g.generateBodySchema(schemaMap.Items)
				return []interface{}{itemSchema}
			}
			return []interface{}{"sample_item"}
		}

		// Handle object type
		if schemaMap.Type != nil && schemaMap.Type.Is("object") {
			result := make(map[string]interface{})
			for key, prop := range schemaMap.Properties {
				result[key] = g.generateBodySchema(prop)
			}
			return result
		}

		// Handle primitive types
		if schemaMap.Type != nil {
			switch {
			case schemaMap.Type.Is("string"):
				if schemaMap.Format != "" {
					switch schemaMap.Format {
					case "email":
						return "test@example.com"
					case "date":
						return "2024-01-01"
					case "date-time":
						return "2024-01-01T12:00:00Z"
					case "uuid":
						return "123e4567-e89b-12d3-a456-426614174000"
					case "uri":
						return "https://example.com"
					case "ipv4":
						return "192.168.1.1"
					case "ipv6":
						return "2001:db8::1"
					}
				}
				if len(schemaMap.Enum) > 0 {
					return schemaMap.Enum[0]
				}
				return "sample_string"
			case schemaMap.Type.Is("number"):
				return 123.45
			case schemaMap.Type.Is("integer"):
				return 123
			case schemaMap.Type.Is("boolean"):
				return true
			}
		}
	}
	return nil
}
