package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"auto-api-tester/internal/types"
)

// TestData represents the test data structure
type TestData struct {
	Endpoints map[string]types.EndpointTestData `json:"endpoints"`
}

// Loader handles loading test data from files
type Loader struct {
	dir string
}

// NewLoader creates a new test data loader
func NewLoader(dir string) *Loader {
	return &Loader{dir: dir}
}

// LoadTestData loads test data from the template file
func (l *Loader) LoadTestData() (*TestData, error) {
	// Try loading from testdata_template.json first
	data, err := l.loadFromFile("testdata_template.json")
	if err != nil {
		// If not found, try testdata.json as fallback
		data, err = l.loadFromFile("testdata.json")
		if err != nil {
			return nil, fmt.Errorf("no test data found: %v", err)
		}
	}
	return data, nil
}

func (l *Loader) loadFromFile(filename string) (*TestData, error) {
	path := filepath.Join(l.dir, filename)
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data TestData
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, fmt.Errorf("failed to parse test data: %v", err)
	}

	return &data, nil
}

// GetTestDataForEndpoint returns test data for a specific endpoint
func (l *Loader) GetTestDataForEndpoint(endpoint types.Endpoint) (*types.EndpointTestData, error) {
	template, err := l.LoadTestData()
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path)
	testData, exists := template.Endpoints[key]
	if !exists {
		return nil, fmt.Errorf("no test data found for endpoint: %s", key)
	}

	return &testData, nil
}
