package reporter

import (
	"encoding/json"
	"fmt"
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
		if result.Error == "" {
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
	// TODO: Implement HTML report generation
	return nil
}
