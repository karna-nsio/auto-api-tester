package parser

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"auto-api-tester/internal/types"

	"github.com/getkin/kin-openapi/openapi3"
)

// SwaggerParser handles parsing of Swagger/OpenAPI specifications
type SwaggerParser struct {
	baseURL string
	client  *http.Client
	doc     *openapi3.T
}

// NewSwaggerParser creates a new instance of SwaggerParser
func NewSwaggerParser(baseURL string) *SwaggerParser {
	return &SwaggerParser{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// ParseEndpoints fetches and parses the Swagger documentation
func (p *SwaggerParser) ParseEndpoints() ([]types.Endpoint, error) {
	// Try different Swagger/OpenAPI JSON URLs
	urls := []string{
		fmt.Sprintf("%s/swagger/v1/swagger.json", p.baseURL),
		fmt.Sprintf("%s/swagger.json", p.baseURL),
		fmt.Sprintf("%s/v1/swagger.json", p.baseURL),
		fmt.Sprintf("%s/api/swagger.json", p.baseURL),
		fmt.Sprintf("%s/api/v1/swagger.json", p.baseURL),
		fmt.Sprintf("%s/swagger/v1/swagger", p.baseURL),
		fmt.Sprintf("%s/swagger", p.baseURL),
	}

	var lastErr error
	for _, url := range urls {
		fmt.Printf("Trying to fetch OpenAPI documentation from: %s\n", url)
		p.doc, lastErr = p.fetchOpenAPIDoc(url)
		if lastErr == nil {
			fmt.Printf("Successfully fetched OpenAPI documentation from: %s\n", url)
			break
		}
		fmt.Printf("Failed to fetch from %s: %v\n", url, lastErr)
	}

	if p.doc == nil {
		return nil, fmt.Errorf("failed to fetch OpenAPI documentation from any known URL. Last error: %v", lastErr)
	}

	return p.extractEndpoints(), nil
}

// fetchOpenAPIDoc fetches the OpenAPI documentation from the given URL
func (p *SwaggerParser) fetchOpenAPIDoc(url string) (*openapi3.T, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI doc: %v", err)
	}

	return doc, nil
}

// extractEndpoints extracts endpoints from the OpenAPI documentation
func (p *SwaggerParser) extractEndpoints() []types.Endpoint {
	var endpoints []types.Endpoint

	paths := p.doc.Paths.Map()
	for path, pathItem := range paths {
		for method, operation := range pathItem.Operations() {
			// Combine base URL with path
			fullPath := p.baseURL + path

			endpoint := types.Endpoint{
				Path:       fullPath,
				Method:     strings.ToUpper(method),
				Parameters: make([]types.Parameter, 0),
				Responses:  make(map[int]types.Response),
			}

			// Extract parameters
			for _, param := range operation.Parameters {
				endpoint.Parameters = append(endpoint.Parameters, types.Parameter{
					Name:     param.Value.Name,
					In:       param.Value.In,
					Required: param.Value.Required,
					Schema:   param.Value.Schema,
				})
			}

			// Extract request body if present
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				// Get the first content type (usually application/json)
				for contentType, content := range operation.RequestBody.Value.Content {
					if content.Schema != nil {
						// Resolve schema reference if present
						schema := content.Schema
						if ref := content.Schema.Ref; ref != "" {
							// Try to resolve the reference
							schemaName := strings.TrimPrefix(ref, "#/components/schemas/")
							if resolved, ok := p.doc.Components.Schemas[schemaName]; ok {
								schema = resolved
							}
						}

						endpoint.Parameters = append(endpoint.Parameters, types.Parameter{
							Name:        "body",
							In:          "body",
							Required:    operation.RequestBody.Value.Required,
							Schema:      schema,
							ContentType: contentType,
						})
						break
					}
				}
			}

			// Extract responses
			responses := operation.Responses.Map()
			for statusCode, response := range responses {
				code := 0
				fmt.Sscanf(statusCode, "%d", &code)
				if code == 0 {
					continue
				}

				description := ""
				if response.Value.Description != nil {
					description = *response.Value.Description
				}

				var schema interface{}
				if content, ok := response.Value.Content["application/json"]; ok && content != nil {
					schema = content.Schema
				}

				endpoint.Responses[code] = types.Response{
					Description: description,
					Schema:      schema,
				}
			}

			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}
