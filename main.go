package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"auto-api-tester/internal/config"
	"auto-api-tester/internal/executor"
	"auto-api-tester/internal/parser"
	"auto-api-tester/internal/reporter"
	"auto-api-tester/internal/testdata"
	"auto-api-tester/internal/testdata/generator"
	"auto-api-tester/internal/types"

	_ "github.com/denisenkom/go-mssqldb" // for sqlserver
	_ "github.com/go-sql-driver/mysql"   // for mysql
	_ "github.com/lib/pq"                // for postgres
)

func convertTestResults(execResults []executor.TestResult) []reporter.TestResult {
	repResults := make([]reporter.TestResult, len(execResults))
	for i, r := range execResults {
		status := 0
		switch r.Status {
		case "SUCCESS":
			// Keep the original status code from the response
			if r.Response == "No Content (204)" {
				status = 204
			} else {
				status = 200
			}
		case "FAILURE":
			status = 400
		case "ERROR":
			status = 500
		}

		// Try to parse response as JSON if it's not empty
		var response interface{}
		if r.Response != "" {
			if err := json.Unmarshal([]byte(r.Response), &response); err != nil {
				// If not JSON, use as string
				response = r.Response
			}
		}

		repResults[i] = reporter.TestResult{
			Endpoint:    r.Endpoint,
			Method:      r.Method,
			Status:      status,
			Duration:    r.Duration,
			Error:       fmt.Sprintf("%v", r.Error),
			RequestBody: r.RequestBody,
			Response:    response,
		}
	}
	return repResults
}

func main() {
	// Check if we're running the generate command with input
	if len(os.Args) > 1 && os.Args[1] == "generate" && len(os.Args) > 2 && os.Args[2] == "--input" {
		// Create a new flag set for the generate command
		generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)

		// Define flags
		dbType := generateCmd.String("db-type", "", "Database type (postgres|mysql|sqlserver)")
		dbHost := generateCmd.String("db-host", "", "Database host")
		dbPort := generateCmd.Int("db-port", 0, "Database port")
		dbName := generateCmd.String("db-name", "", "Database name")
		dbUser := generateCmd.String("db-user", "", "Database user")
		dbPassword := generateCmd.String("db-password", "", "Database password")
		templatePath := generateCmd.String("template", "", "Path to testdata template file")
		outputPath := generateCmd.String("output", "", "Path to output testdata file")

		// Parse flags
		if err := generateCmd.Parse(os.Args[3:]); err != nil {
			log.Fatalf("Failed to parse flags: %v", err)
		}

		// Validate required flags
		if *dbType == "" || *dbHost == "" || *dbPort == 0 || *dbName == "" || *dbUser == "" || *dbPassword == "" {
			fmt.Println("Error: All database configuration flags are required")
			generateCmd.Usage()
			os.Exit(1)
		}

		if *templatePath == "" || *outputPath == "" {
			fmt.Println("Error: Template and output paths are required")
			generateCmd.Usage()
			os.Exit(1)
		}

		// Create database configuration
		dbConfig := generator.DBConfig{
			Type:     *dbType,
			Host:     *dbHost,
			Port:     *dbPort,
			Database: *dbName,
			User:     *dbUser,
			Password: *dbPassword,
		}

		// Initialize database generator
		dbGenerator := generator.NewDBGenerator(dbConfig, *templatePath, *outputPath)

		// Generate test data
		if err := dbGenerator.GenerateTestData(); err != nil {
			log.Fatalf("Failed to generate test data: %v", err)
		}

		fmt.Printf("Test data generated successfully in %s\n", *outputPath)
		return
	}

	// Check if we're running the generate command
	if len(os.Args) > 1 && os.Args[1] == "generate" {
		// Run the generate command
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		return
	}

	// Check if we're running the generate command with URL
	if len(os.Args) > 1 && os.Args[1] == "-url" {
		// This is the generate command
		swaggerURL := os.Args[2]
		outputDir := "testdata"
		if len(os.Args) > 3 && os.Args[3] == "-output" {
			outputDir = os.Args[4]
		}

		// Initialize Swagger parser
		swaggerParser := parser.NewSwaggerParser(swaggerURL)

		// Parse endpoints
		endpoints, err := swaggerParser.ParseEndpoints()
		if err != nil {
			log.Fatalf("Failed to parse endpoints: %v", err)
		}

		fmt.Printf("Found %d endpoints to test\n", len(endpoints))

		// Generate test data template
		testDataGenerator := testdata.NewGenerator(outputDir)
		if err := testDataGenerator.GenerateTemplate(endpoints); err != nil {
			log.Fatalf("Failed to generate test data template: %v", err)
		}

		fmt.Printf("Test data template generated successfully in %s/testdata_template.json\n", outputDir)
		fmt.Println("Please review and modify the template as needed, then rename it to testdata.json to run the tests.")
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Load test data
	testDataLoader := testdata.NewLoader("testdata")
	testData, err := testDataLoader.LoadTestData()
	if err != nil {
		fmt.Println("No test data found. Please generate test data template first:")
		fmt.Println("  auto-api-tester generate -url <swagger-url>")
		fmt.Println("Then fill in the test data in testdata/testdata_template.json")
		return
	}

	// Convert test data to endpoints
	endpoints := make([]types.Endpoint, 0)
	for endpoint, data := range testData.Endpoints {
		// Parse method and path from endpoint string (e.g., "GET /api/users")
		parts := strings.SplitN(endpoint, " ", 2)
		if len(parts) != 2 {
			continue
		}
		method := parts[0]
		path := parts[1]

		// Create endpoint with test data
		ep := types.Endpoint{
			Method: method,
			Path:   path,
			TestData: types.EndpointTestData{
				PathParams:  data.PathParams,
				QueryParams: data.QueryParams,
				Body:        data.Body,
				Headers:     data.Headers,
			},
		}
		endpoints = append(endpoints, ep)
	}

	fmt.Printf("Loaded %d endpoints from test data\n", len(endpoints))

	// Initialize test executor
	testExecutor := executor.NewTestExecutor(executor.TestConfig{
		Concurrent: cfg.Test.Concurrent,
		MaxWorkers: cfg.Test.MaxWorkers,
		Timeout:    cfg.Test.Timeout,
		Retry: executor.RetryConfig{
			Attempts: cfg.Test.Retry.Attempts,
			Delay:    time.Duration(cfg.Test.Retry.Delay) * time.Second,
		},
	}, testDataLoader)

	// Initialize reporter
	testReporter := reporter.NewReporter(reporter.ReportingConfig{
		Format:    []string{cfg.Reporting.Format},
		OutputDir: cfg.Reporting.OutputDir,
		Detailed:  cfg.Reporting.Detailed,
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Test.Timeout)*time.Second)
	defer cancel()

	// Run tests
	results := testExecutor.RunTests(ctx, endpoints)

	// Generate report
	if err := testReporter.GenerateReport(convertTestResults(results)); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	fmt.Println("API testing completed successfully!")
}
