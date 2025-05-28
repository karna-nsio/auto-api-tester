package reporter

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"time"
)

// Report represents the test execution report
type Report struct {
	Timestamp   time.Time
	TotalTests  int
	PassedTests int
	FailedTests int
	Duration    time.Duration
	Results     []TestResult
}

// TestResult represents a single test result
type TestResult struct {
	Endpoint    string
	Method      string
	Status      int
	Duration    time.Duration
	Error       string
	RequestBody interface{}
	Response    interface{}
}

// Reporter handles the generation of test reports
type Reporter struct {
	config ReportingConfig
}

// ReportingConfig holds the configuration for reporting
type ReportingConfig struct {
	Format    []string
	OutputDir string
	Detailed  bool
}

// NewReporter creates a new instance of Reporter
func NewReporter(config ReportingConfig) *Reporter {
	return &Reporter{
		config: config,
	}
}

// GenerateReport generates the test execution report
func (r *Reporter) GenerateReport(results []TestResult) error {
	report := Report{
		Timestamp:   time.Now(),
		TotalTests:  len(results),
		PassedTests: 0,
		FailedTests: 0,
		Results:     results,
	}

	// Calculate passed and failed tests
	for _, result := range results {
		if result.Status >= 200 && result.Status < 300 {
			report.PassedTests++
		} else {
			report.FailedTests++
		}
	}

	// Generate reports in specified formats
	for _, format := range r.config.Format {
		switch format {
		case "json":
			if err := r.generateJSONReport(report); err != nil {
				return fmt.Errorf("failed to generate JSON report: %v", err)
			}
		case "html":
			if err := r.generateHTMLReport(report); err != nil {
				return fmt.Errorf("failed to generate HTML report: %v", err)
			}
		}
	}

	return nil
}

// generateJSONReport generates a JSON format report
func (r *Reporter) generateJSONReport(report Report) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(r.config.OutputDir, 0755); err != nil {
		return err
	}

	// Generate report file path
	reportPath := filepath.Join(r.config.OutputDir, fmt.Sprintf("report_%s.json", report.Timestamp.Format("20060102_150405")))

	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	// Write report to file
	return os.WriteFile(reportPath, data, 0644)
}

// generateHTMLReport generates an HTML format report
func (r *Reporter) generateHTMLReport(report Report) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(r.config.OutputDir, 0755); err != nil {
		return err
	}

	// Generate report file path
	reportPath := filepath.Join(r.config.OutputDir, fmt.Sprintf("report_%s.html", report.Timestamp.Format("20060102_150405")))

	// Create HTML content
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Test Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background-color: #f8f9fa;
            padding: 20px;
            border-radius: 6px;
            text-align: center;
        }
        .summary-card h3 {
            margin: 0;
            color: #666;
        }
        .summary-card .number {
            font-size: 2em;
            font-weight: bold;
            margin: 10px 0;
        }
        .passed { color: #28a745; }
        .failed { color: #dc3545; }
        .total { color: #007bff; }
        .results {
            margin-top: 30px;
        }
        .test-case {
            background-color: #fff;
            border: 1px solid #dee2e6;
            border-radius: 6px;
            margin-bottom: 15px;
            padding: 15px;
        }
        .test-case.passed {
            border-left: 4px solid #28a745;
        }
        .test-case.failed {
            border-left: 4px solid #dc3545;
        }
        .test-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 10px;
        }
        .test-details {
            background-color: #f8f9fa;
            padding: 10px;
            border-radius: 4px;
            margin-top: 10px;
        }
        .timestamp {
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>API Test Report</h1>
            <p class="timestamp">Generated on: %s</p>
        </div>
        
        <div class="summary">
            <div class="summary-card">
                <h3>Total Tests</h3>
                <div class="number total">%d</div>
            </div>
            <div class="summary-card">
                <h3>Passed Tests</h3>
                <div class="number passed">%d</div>
            </div>
            <div class="summary-card">
                <h3>Failed Tests</h3>
                <div class="number failed">%d</div>
            </div>
            <div class="summary-card">
                <h3>Duration</h3>
                <div class="number">%s</div>
            </div>
        </div>

        <div class="results">
            <h2>Test Results</h2>`,
		report.Timestamp.Format("2006-01-02 15:04:05"),
		report.TotalTests,
		report.PassedTests,
		report.FailedTests,
		report.Duration.Round(time.Millisecond))

	// Add test results
	for _, result := range report.Results {
		statusClass := "passed"
		// A test is considered failed if:
		// 1. There is an error message OR
		// 2. The status code is not in the 2xx range
		if result.Error != "" || result.Status < 200 || result.Status >= 300 {
			statusClass = "failed"
		}

		htmlContent += fmt.Sprintf(`
            <div class="test-case %s">
                <div class="test-header">
                    <strong>%s %s</strong>
                    <span>Status: %d</span>
                </div>
                <div>Duration: %s</div>`,
			statusClass,
			result.Method,
			result.Endpoint,
			result.Status,
			result.Duration.Round(time.Millisecond))

		// Only show error message if there is one
		if result.Error != "" {
			htmlContent += fmt.Sprintf(`
                <div class="test-details">
                    <strong>Error:</strong> %s
                </div>`, result.Error)
		}

		if r.config.Detailed {
			requestBody, _ := json.MarshalIndent(result.RequestBody, "", "  ")
			response, _ := json.MarshalIndent(result.Response, "", "  ")

			htmlContent += fmt.Sprintf(`
                <div class="test-details">
                    <strong>Request Body:</strong>
                    <pre>%s</pre>
                    <strong>Response:</strong>
                    <pre>%s</pre>
                </div>`,
				html.EscapeString(string(requestBody)),
				html.EscapeString(string(response)))
		}

		htmlContent += `
            </div>`
	}

	htmlContent += `
        </div>
    </div>
</body>
</html>`

	// Write report to file
	return os.WriteFile(reportPath, []byte(htmlContent), 0644)
}
