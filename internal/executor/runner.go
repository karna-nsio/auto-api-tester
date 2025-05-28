package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"auto-api-tester/internal/testdata"
	"auto-api-tester/internal/types"
)

// TestResult represents the result of a single test
type TestResult struct {
	Endpoint    string
	Method      string
	Status      string
	Duration    time.Duration
	Error       error
	RequestBody string
	Response    string
}

// TestConfig holds configuration for test execution
type TestConfig struct {
	Concurrent bool
	MaxWorkers int
	Timeout    int
	Retry      RetryConfig
}

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	Attempts int
	Delay    time.Duration
}

// TestExecutor handles the execution of API tests
type TestExecutor struct {
	config   TestConfig
	client   *http.Client
	testData *testdata.Loader
}

// NewTestExecutor creates a new test executor
func NewTestExecutor(config TestConfig, testData *testdata.Loader) *TestExecutor {
	return &TestExecutor{
		config:   config,
		client:   &http.Client{Timeout: time.Duration(config.Timeout) * time.Second},
		testData: testData,
	}
}

// RunTests executes tests for all endpoints
func (e *TestExecutor) RunTests(ctx context.Context, endpoints []types.Endpoint) []TestResult {
	var results []TestResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create a channel to limit concurrent executions
	sem := make(chan struct{}, e.config.MaxWorkers)

	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(endpoint types.Endpoint) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Get test data for this endpoint
			testData, err := e.testData.GetTestDataForEndpoint(endpoint)
			if err != nil {
				mu.Lock()
				results = append(results, TestResult{
					Endpoint: endpoint.Path,
					Method:   endpoint.Method,
					Status:   "ERROR",
					Error:    fmt.Errorf("failed to get test data: %w", err),
				})
				mu.Unlock()
				return
			}

			// Build request
			req, err := e.buildRequest(ctx, endpoint, testData)
			if err != nil {
				mu.Lock()
				results = append(results, TestResult{
					Endpoint: endpoint.Path,
					Method:   endpoint.Method,
					Status:   "ERROR",
					Error:    fmt.Errorf("failed to build request: %w", err),
				})
				mu.Unlock()
				return
			}

			// Execute test with retries
			var result TestResult
			for attempt := 0; attempt < e.config.Retry.Attempts; attempt++ {
				result = e.executeTest(req, endpoint)
				if result.Error == nil {
					break
				}
				time.Sleep(e.config.Retry.Delay)
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(endpoint)
	}

	wg.Wait()
	return results
}

// buildRequest creates an HTTP request for the given endpoint and test data
func (e *TestExecutor) buildRequest(ctx context.Context, endpoint types.Endpoint, testData *types.EndpointTestData) (*http.Request, error) {
	// Replace path parameters
	url := endpoint.Path
	for key, value := range testData.PathParams {
		url = strings.Replace(url, fmt.Sprintf("{%s}", key), fmt.Sprint(value), -1)
	}

	// Add query parameters
	if len(testData.QueryParams) > 0 {
		query := make([]string, 0, len(testData.QueryParams))
		for key, value := range testData.QueryParams {
			query = append(query, fmt.Sprintf("%s=%s", key, fmt.Sprint(value)))
		}
		url = fmt.Sprintf("%s?%s", url, strings.Join(query, "&"))
	}

	// Debug logging for request
	fmt.Printf("Request URL: %s\n", url)
	fmt.Printf("Request Method: %s\n", endpoint.Method)
	fmt.Printf("Request Headers: %v\n", testData.Headers)
	if testData.Body != nil {
		bodyBytes, _ := json.Marshal(testData.Body)
		fmt.Printf("Request Body: %s\n", string(bodyBytes))
	}

	// Create request
	var body io.Reader
	if testData.Body != nil {
		bodyBytes, err := json.Marshal(testData.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range testData.Headers {
		req.Header.Set(key, fmt.Sprint(value))
	}

	return req, nil
}

// executeTest executes a single test and returns the result
func (e *TestExecutor) executeTest(req *http.Request, endpoint types.Endpoint) TestResult {
	start := time.Now()
	resp, err := e.client.Do(req)
	duration := time.Since(start)

	result := TestResult{
		Endpoint: endpoint.Path,
		Method:   endpoint.Method,
		Duration: duration,
	}

	if err != nil {
		result.Status = "ERROR"
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = "ERROR"
		result.Error = fmt.Errorf("failed to read response body: %w", err)
		return result
	}

	// Debug logging
	fmt.Printf("Response Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Raw Response Body: %s\n", string(body))

	// Set result status based on response status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = "SUCCESS"
	} else {
		result.Status = "FAILURE"
		result.Error = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Format response body if it's JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonResponse interface{}
		if err := json.Unmarshal(body, &jsonResponse); err == nil {
			// Pretty print the JSON response
			if prettyJSON, err := json.MarshalIndent(jsonResponse, "", "  "); err == nil {
				result.Response = string(prettyJSON)
				fmt.Printf("Formatted JSON Response: %s\n", result.Response)
			} else {
				result.Response = string(body)
				fmt.Printf("Failed to format JSON, using raw response: %s\n", result.Response)
			}
		} else {
			result.Response = string(body)
			fmt.Printf("Failed to parse JSON, using raw response: %s\n", result.Response)
		}
	} else {
		result.Response = string(body)
		fmt.Printf("Non-JSON response: %s\n", result.Response)
	}

	return result
}

// Endpoint represents an API endpoint to test
type Endpoint struct {
	Path       string
	Method     string
	Parameters []Parameter
	Responses  map[int]Response
}

// Parameter represents an API parameter
type Parameter struct {
	Name     string
	In       string
	Required bool
	Schema   interface{}
}

// Response represents an API response
type Response struct {
	Description string
	Schema      interface{}
}
