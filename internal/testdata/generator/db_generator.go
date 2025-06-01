package generator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb" // for sqlserver
	_ "github.com/go-sql-driver/mysql"   // for mysql
	_ "github.com/lib/pq"                // for postgres

	"auto-api-tester/internal/llm"
	"auto-api-tester/internal/logger"
	"auto-api-tester/internal/types"

	"github.com/google/uuid"
)

// DBConfig holds database connection configuration
type DBConfig struct {
	Type     string
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// DBGenerator handles test data generation from database
type DBGenerator struct {
	config       DBConfig
	db           *sql.DB
	templatePath string
	outputPath   string
	analyzer     *TableAnalyzer
	llmClient    llm.LLMClient
}

// NewDBGenerator creates a new instance of DBGenerator
func NewDBGenerator(dbConfig DBConfig, llmConfig llm.Config, templatePath, outputPath string) *DBGenerator {
	// Initialize random number generator
	rand.Seed(time.Now().UnixNano())

	logger, _ := logger.NewLogger("db_generator")

	llmClient, _ := llm.NewClient(&llmConfig, logger)

	return &DBGenerator{
		config:       dbConfig,
		templatePath: templatePath,
		outputPath:   outputPath,
		llmClient:    llmClient,
	}
}

// GenerateTestData generates test data using database information
func (g *DBGenerator) GenerateTestData() error {
	// 1. Connect to database
	if err := g.connect(); err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer g.db.Close()

	// 2. Initialize table analyzer
	g.analyzer = NewTableAnalyzer(g.db)

	// 3. Load template
	template, err := g.loadTemplate()
	if err != nil {
		return fmt.Errorf("failed to load template: %v", err)
	}

	// 4. Generate test data for each endpoint
	for endpoint, data := range template.Endpoints {
		// Parse endpoint string (e.g., "GET /api/users")
		method, path := parseEndpointString(endpoint)

		// Generate test data based on endpoint type and database schema
		testData, err := g.generateEndpointData(method, path, data)
		if err != nil {
			fmt.Printf("Warning: Failed to generate test data for %s: %v\n", endpoint, err)
			continue
		}

		// Update template with generated data
		template.Endpoints[endpoint] = testData
	}

	// 5. Save generated test data
	return g.saveTestData(template)
}

// connect establishes database connection
func (g *DBGenerator) connect() error {
	var dsn string
	switch g.config.Type {
	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			g.config.Host, g.config.Port, g.config.User, g.config.Password, g.config.Database)
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			g.config.User, g.config.Password, g.config.Host, g.config.Port, g.config.Database)
	case "sqlserver":
		dsn = fmt.Sprintf("server=%s;port=%d;user id=%s;password=%s;database=%s",
			g.config.Host, g.config.Port, g.config.User, g.config.Password, g.config.Database)
	default:
		return fmt.Errorf("unsupported database type: %s", g.config.Type)
	}

	db, err := sql.Open(g.config.Type, dsn)
	if err != nil {
		return err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return err
	}

	g.db = db
	return nil
}

// loadTemplate loads the test data template
func (g *DBGenerator) loadTemplate() (*types.TestDataTemplate, error) {
	data, err := os.ReadFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %v", err)
	}

	var template types.TestDataTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template file: %v", err)
	}

	return &template, nil
}

// saveTestData saves the generated test data
func (g *DBGenerator) saveTestData(template *types.TestDataTemplate) error {
	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(g.outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Marshal template to JSON
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal test data: %v", err)
	}

	// Write to file
	if err := os.WriteFile(g.outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write test data file: %v", err)
	}

	return nil
}

// parseEndpointString parses an endpoint string into method and path
func parseEndpointString(endpoint string) (string, string) {
	parts := strings.SplitN(endpoint, " ", 2)
	if len(parts) != 2 {
		return "", endpoint
	}
	return parts[0], parts[1]
}

// generateEndpointData generates test data for a specific endpoint
func (g *DBGenerator) generateEndpointData(method, path string, data types.EndpointTestData) (types.EndpointTestData, error) {
	// Create a copy of template data
	testData := data

	// Analyze endpoint to determine related database tables
	tables, err := g.analyzeEndpointTables(method, path)
	if err != nil {
		return testData, err
	}

	// Get a sample record from the main table

	fmt.Println("tables[0]", tables[0])
	sampleRecord, err := g.getSampleRecord(tables[0])
	if err != nil {
		return testData, fmt.Errorf("failed to get sample record: %v", err)
	}

	// Generate data based on HTTP method and database tables
	switch method {
	case "GET":
		return g.generateGetData(path, testData, tables, sampleRecord)
	case "POST":
		return g.generatePostData(path, testData, tables, sampleRecord)
	case "PUT":
		return g.generatePutData(path, testData, tables, sampleRecord)
	case "DELETE":
		return g.generateDeleteData(path, testData, tables, sampleRecord)
	default:
		return testData, fmt.Errorf("unsupported HTTP method: %s", method)
	}
}

// analyzeEndpointTables determines which database tables are related to the endpoint
func (g *DBGenerator) analyzeEndpointTables(method, path string) ([]string, error) {
	// Extract table name from path (e.g., /api/users -> users)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	// Get the last part of the path as the potential table name
	tableName := strings.ToLower(parts[len(parts)-1])

	// Query to get the actual table name from database
	checkQuery := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE LOWER(table_name) = LOWER($1)
		LIMIT 1
	`
	var actualTableName string
	err := g.db.QueryRow(checkQuery, tableName).Scan(&actualTableName)
	if err != nil {
		if err == sql.ErrNoRows {
			// Table not found, use LLM to suggest alternatives
			if g.llmClient == nil {
				return nil, fmt.Errorf("table '%s' not found and LLM client is not available", tableName)
			}

			fmt.Printf("Table '%s' not found. Using LLM to suggest alternatives...\n", tableName)

			// Get schema information for LLM analysis
			schemaInfo := g.getSchemaInfo()

			// Use LLM to analyze relationships and suggest similar tables
			analysis, err := g.llmClient.AnalyzeRelationships(context.Background(), tableName, schemaInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to analyze relationships with LLM: %v", err)
			}

			// Present suggestions to user with more details
			fmt.Printf("\nSuggested tables for endpoint %s %s:\n", method, path)

			// Display similar tables
			fmt.Println("\nSimilar tables found:")
			for i, similar := range analysis.SimilarTables {
				fmt.Printf("%d. %s and %s\n", i+1, similar.Table1, similar.Table2)
				fmt.Printf("   Reason: %s\n", similar.Reasoning)
			}

			// Display foreign key relationships
			fmt.Println("\nForeign key relationships:")
			for i, fk := range analysis.ForeignKeysAndDependencies {
				fmt.Printf("%d. %s.%s -> %s.%s\n", i+1,
					fk.Table, fk.ForeignKey,
					fk.References.Table, fk.References.Column)
			}

			fmt.Printf("\n0. Enter custom table name\n")

			// Get user input
			var choice int
			fmt.Print("\nSelect a table (enter number): ")
			fmt.Scanln(&choice)

			if choice == 0 {
				// Get custom table name
				fmt.Print("Enter custom table name: ")
				fmt.Scanln(&tableName)
			} else if choice > 0 && choice <= len(analysis.SimilarTables) {
				// Use the first table from the selected similar tables pair
				tableName = analysis.SimilarTables[choice-1].Table1
			} else {
				return nil, fmt.Errorf("invalid selection")
			}
		} else {
			return nil, fmt.Errorf("failed to query table: %v", err)
		}
	}
	// Find related tables
	relatedTables, err := g.analyzer.FindRelatedTables(tableName)
	if err != nil {
		return nil, err
	}

	fmt.Println("actualTableName: ", actualTableName)
	// Add the main table to the list
	tables := append([]string{actualTableName}, relatedTables...)
	return tables, nil
}

// getSchemaInfo returns schema information for LLM analysis
func (g *DBGenerator) getSchemaInfo() map[string]interface{} {
	schemaInfo := make(map[string]interface{})

	// Query to get tables with a limit to reduce token usage
	rows, err := g.db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
		LIMIT 10  -- Limit to most relevant tables
	`)
	if err != nil {
		return schemaInfo
	}
	defer rows.Close()

	// Get table information
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		// Get only essential columns for each table
		colRows, err := g.db.Query(`
			SELECT column_name, data_type
			FROM information_schema.columns
			WHERE table_name = $1
			AND column_name IN (
				SELECT column_name 
				FROM information_schema.key_column_usage 
				WHERE table_name = $1
				UNION
				SELECT column_name 
				FROM information_schema.constraint_column_usage 
				WHERE table_name = $1
			)
		`, tableName)
		if err != nil {
			continue
		}

		columns := make([]map[string]string, 0)
		for colRows.Next() {
			var colName, dataType string
			if err := colRows.Scan(&colName, &dataType); err != nil {
				continue
			}
			columns = append(columns, map[string]string{
				"name": colName,
				"type": dataType,
			})
		}
		colRows.Close()

		if len(columns) > 0 {
			schemaInfo[tableName] = columns
		}
	}

	return schemaInfo
}

// getSampleRecord retrieves a random record from the specified table
func (g *DBGenerator) getSampleRecord(tableName string) (map[string]interface{}, error) {
	// Get table structure
	tableInfo, err := g.analyzer.analyzeTable(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze table %s: %v", tableName, err)
	}
	fmt.Println("tableInfo: ", tableInfo)

	// Build SELECT query with all columns
	columns := make([]string, len(tableInfo.Columns))
	for i, col := range tableInfo.Columns {
		// Quote column names to handle case sensitivity
		columns[i] = fmt.Sprintf(`"%s"`, col.Name)
	}
	fmt.Println("table name in getSampleRecord", tableName)
	// Quote the table name to handle case sensitivity
	query := fmt.Sprintf(`SELECT %s FROM "%s" ORDER BY RANDOM() LIMIT 1`,
		strings.Join(columns, ", "), tableName)

	// Execute query
	rows, err := g.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table %s: %v", tableName, err)
	}
	defer rows.Close()

	// Get column names
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %v", err)
	}

	// Prepare slice for row values
	values := make([]interface{}, len(columnNames))
	valuePtrs := make([]interface{}, len(columnNames))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the row
	if !rows.Next() {
		return nil, fmt.Errorf("no records found in table %s", tableName)
	}
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, fmt.Errorf("failed to scan row: %v", err)
	}

	// Convert row to map
	record := make(map[string]interface{})
	for i, col := range columnNames {
		val := values[i]
		if val != nil {
			record[col] = val
		}
	}

	return record, nil
}

// generateGetData generates test data for GET endpoints
func (g *DBGenerator) generateGetData(path string, data types.EndpointTestData, tables []string, sampleRecord map[string]interface{}) (types.EndpointTestData, error) {
	// Use LLM to analyze the sample record and generate appropriate query parameters
	if g.llmClient != nil {
		analysis, err := g.llmClient.AnalyzeBusinessRules(context.Background(), tables[0], []map[string]interface{}{sampleRecord})
		if err != nil {
			return data, fmt.Errorf("failed to analyze sample record: %v", err)
		}

		// Generate query parameters based on the analysis
		if len(data.QueryParams) > 0 {
			for param, paramValue := range data.QueryParams {
				if paramValue == nil {
					// Use LLM to generate appropriate value based on sample record
					generatedValue, err := g.generateValueFromSample(param, sampleRecord, analysis)
					if err != nil {
						return data, err
					}
					data.QueryParams[param] = generatedValue
				}
			}
		}

		// Generate path parameters based on the analysis
		if len(data.PathParams) > 0 {
			for param, paramValue := range data.PathParams {
				if paramValue == nil {
					// Use LLM to generate appropriate value based on sample record
					generatedValue, err := g.generateValueFromSample(param, sampleRecord, analysis)
					if err != nil {
						return data, err
					}
					data.PathParams[param] = generatedValue
				}
			}
		}
	}

	return data, nil
}

// generatePostData generates test data for POST endpoints
func (g *DBGenerator) generatePostData(path string, data types.EndpointTestData, tables []string, sampleRecord map[string]interface{}) (types.EndpointTestData, error) {
	// Use LLM to analyze the sample record and generate appropriate request body
	if g.llmClient != nil {
		// Prepare the context for LLM analysis
		llmContext := map[string]interface{}{
			"endpoint": map[string]interface{}{
				"method": "POST",
				"path":   path,
				"body":   data.Body, // Pass the original body template
			},
			"sampleRecord": sampleRecord,
			"table":        tables[0],
		}

		// Use LLM to analyze and generate data
		analysis, err := g.llmClient.AnalyzeBusinessRules(context.Background(), tables[0], []map[string]interface{}{llmContext})
		if err != nil {
			return data, fmt.Errorf("failed to analyze sample record: %v", err)
		}

		// Generate request body based on the analysis, sample record, and endpoint template
		// generatedBody, err := g.generateBodyFromTemplate(data.Body, sampleRecord, analysis)
		// if err != nil {
		// 	return data, err
		// }
		data.Body = analysis
	}

	return data, nil
}

// generateBodyFromTemplate generates a request body based on the template, sample record, and analysis
func (g *DBGenerator) generateBodyFromTemplate(template interface{}, sampleRecord map[string]interface{}, analysis *llm.AnalysisResult) (interface{}, error) {
	switch t := template.(type) {
	case map[string]interface{}:
		// Handle object template
		return g.generateObjectFromTemplate(t, sampleRecord, analysis)
	case []interface{}:
		// Handle array template
		return g.generateArrayFromTemplate(t, sampleRecord, analysis)
	default:
		// If template is nil or not a map/array, use sample record directly
		return g.generateBodyFromSample(sampleRecord, analysis)
	}
}

// generateObjectFromTemplate generates an object based on the template structure
func (g *DBGenerator) generateObjectFromTemplate(template map[string]interface{}, sampleRecord map[string]interface{}, analysis *llm.AnalysisResult) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Process each field in the template
	for field, templateValue := range template {
		// If template has a specific value, use it
		if templateValue != nil {
			// Check if the value is a nested object or array
			switch v := templateValue.(type) {
			case map[string]interface{}:
				// Recursively generate nested object
				nestedObj, err := g.generateObjectFromTemplate(v, sampleRecord, analysis)
				if err != nil {
					return nil, err
				}
				result[field] = nestedObj
			case []interface{}:
				// Generate array
				nestedArr, err := g.generateArrayFromTemplate(v, sampleRecord, analysis)
				if err != nil {
					return nil, err
				}
				result[field] = nestedArr
			default:
				// Use template value directly
				result[field] = templateValue
			}
			continue
		}

		// If template value is nil, generate new value based on sample record
		value, err := g.generateValueFromSample(field, sampleRecord, analysis)
		if err != nil {
			return nil, err
		}
		result[field] = value
	}

	return result, nil
}

// generateArrayFromTemplate generates an array based on the template structure
func (g *DBGenerator) generateArrayFromTemplate(template []interface{}, sampleRecord map[string]interface{}, analysis *llm.AnalysisResult) ([]interface{}, error) {
	if len(template) == 0 {
		// If template is empty, generate a single item based on sample record
		item, err := g.generateBodyFromSample(sampleRecord, analysis)
		if err != nil {
			return nil, err
		}
		return []interface{}{item}, nil
	}

	// Use first item in template as the structure
	templateItem := template[0]
	result := make([]interface{}, 0)

	// Generate 1-3 items based on the template structure
	numItems := rand.Intn(3) + 1
	for i := 0; i < numItems; i++ {
		var item interface{}
		var err error

		switch v := templateItem.(type) {
		case map[string]interface{}:
			item, err = g.generateObjectFromTemplate(v, sampleRecord, analysis)
		case []interface{}:
			item, err = g.generateArrayFromTemplate(v, sampleRecord, analysis)
		default:
			item, err = g.generateValueFromSample("", sampleRecord, analysis)
		}

		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, nil
}

// generatePutData generates test data for PUT endpoints
func (g *DBGenerator) generatePutData(path string, data types.EndpointTestData, tables []string, sampleRecord map[string]interface{}) (types.EndpointTestData, error) {
	// Similar to POST, but we need to ensure we have an ID
	return g.generatePostData(path, data, tables, sampleRecord)
}

// generateDeleteData generates test data for DELETE endpoints
func (g *DBGenerator) generateDeleteData(path string, data types.EndpointTestData, tables []string, sampleRecord map[string]interface{}) (types.EndpointTestData, error) {
	// Similar to GET, but we only need the ID
	return g.generateGetData(path, data, tables, sampleRecord)
}

// generateValueFromSample generates a value based on the sample record and analysis
func (g *DBGenerator) generateValueFromSample(param string, sampleRecord map[string]interface{}, analysis interface{}) (interface{}, error) {
	// First try to find a matching field in the sample record
	// if value, exists := sampleRecord[param]; exists {
	// 	return value, nil
	// }

	// // If no exact match, try to find similar fields
	// for field, fieldValue := range sampleRecord {
	// 	if strings.Contains(strings.ToLower(field), strings.ToLower(param)) {
	// 		// Found a similar field, return its value
	// 		return fieldValue, nil
	// 	}
	// }

	// If no similar fields found, use the original value generation logic
	return nil, nil
}

// generateBodyFromSample generates a request body based on the sample record and analysis
func (g *DBGenerator) generateBodyFromSample(sampleRecord map[string]interface{}, analysis *llm.AnalysisResult) (interface{}, error) {
	// Create a new map for the generated body
	// generatedBody := make(map[string]interface{})

	// // Copy values from sample record, but modify them slightly
	// for field, _ := range sampleRecord {
	// 	// Skip auto-increment primary keys
	// 	if strings.HasSuffix(strings.ToLower(field), "id") && g.isAutoIncrement(field) {
	// 		continue
	// 	}

	// 	// Generate a new value based on the type
	// 	newValue, err := g.generateValueFromSample(field, sampleRecord, analysis)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	generatedBody[field] = newValue
	// }

	return nil, nil
}

// isAutoIncrement checks if a field is likely an auto-increment primary key
func (g *DBGenerator) isAutoIncrement(field string) bool {
	return strings.HasSuffix(strings.ToLower(field), "id") &&
		(strings.HasPrefix(strings.ToLower(field), "id") ||
			strings.Contains(strings.ToLower(field), "_id"))
}

// generateValueFromDB generates a value from database
func (g *DBGenerator) generateValueFromDB(param string, tables []string) (interface{}, error) {
	// First, try to find the column in the tables
	for _, table := range tables {
		// Get table info
		tableInfo, err := g.analyzer.analyzeTable(table)
		if err != nil {
			continue
		}

		// Look for a matching column
		for _, col := range tableInfo.Columns {
			if strings.EqualFold(col.Name, param) {
				// Found a matching column, generate a value based on its type
				return g.generateValueForType(col.Type, col.Nullable, col.Name, col)
			}
		}
	}

	// If no matching column found, use LLM to suggest value generation
	if g.llmClient == nil {
		return nil, fmt.Errorf("no matching column found for '%s' and LLM client is not available", param)
	}

	fmt.Printf("No matching column found for '%s'. Using LLM to suggest value...\n", param)

	// Use LLM to analyze the parameter and suggest a value
	analysis, err := g.llmClient.AnalyzeColumn(context.Background(), "", param, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze parameter with LLM: %v", err)
	}

	// Present suggestions to user
	fmt.Printf("\nSuggested value types for '%s':\n", param)
	fmt.Printf("1. %s\n", analysis.DataPatterns.DataType)
	if len(analysis.DataPatterns.ValueRange) > 0 {
		fmt.Printf("2. Use one of these values: %v\n", analysis.DataPatterns.ValueRange)
	}
	fmt.Printf("3. Enter custom value\n")

	// Get user input
	var choice int
	fmt.Print("\nSelect an option (enter number): ")
	fmt.Scanln(&choice)

	var value interface{}
	switch choice {
	case 1:
		// Generate value based on suggested type
		value, err = g.generateValueForType(analysis.DataPatterns.DataType, true, param, ColumnInfo{})
	case 2:
		if len(analysis.DataPatterns.ValueRange) > 0 {
			// Use a random value from the range
			value = analysis.DataPatterns.ValueRange[rand.Intn(len(analysis.DataPatterns.ValueRange))]
		} else {
			value, err = g.generateValueForType(analysis.DataPatterns.DataType, true, param, ColumnInfo{})
		}
	case 3:
		// Get custom value
		fmt.Print("Enter custom value: ")
		var customValue string
		fmt.Scanln(&customValue)
		value = customValue
	default:
		return nil, fmt.Errorf("invalid selection")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate value: %v", err)
	}

	return value, nil
}

// generateBodyFromDB generates body data from database tables
func (g *DBGenerator) generateBodyFromDB(tables []string) (interface{}, error) {
	if len(tables) == 0 {
		return nil, nil
	}

	// Use the first table as the main table
	mainTable := tables[0]
	tableInfo, err := g.analyzer.analyzeTable(mainTable)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze table %s: %v", mainTable, err)
	}

	// Load the template to get the fields we need to generate
	template, err := g.loadTemplate()
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %v", err)
	}

	// Create a map to hold the generated data
	data := make(map[string]interface{})

	// Get the template fields for this endpoint
	var templateFields map[string]interface{}
	for endpoint, endpointData := range template.Endpoints {
		// Extract the path from the endpoint string (e.g., "POST http://localhost:8080/Customer" -> "Customer")
		endpointParts := strings.Split(endpoint, " ")
		if len(endpointParts) < 2 {
			continue
		}
		path := endpointParts[len(endpointParts)-1]
		pathParts := strings.Split(path, "/")
		endpointTable := strings.ToLower(pathParts[len(pathParts)-1])

		// Compare the endpoint table name with the main table name (both in lowercase)
		if endpointTable == strings.ToLower(mainTable) {
			// Handle both array and object body formats
			switch body := endpointData.Body.(type) {
			case map[string]interface{}:
				templateFields = body
			case []interface{}:
				if len(body) > 0 {
					if obj, ok := body[0].(map[string]interface{}); ok {
						templateFields = obj
					}
				}
			}
			break
		}
	}

	// If no template fields found, return empty data
	if templateFields == nil {
		return data, nil
	}

	// Generate values only for fields present in the template
	for fieldName, defaultValue := range templateFields {
		// Find the column in the table
		var col *ColumnInfo
		for _, c := range tableInfo.Columns {
			if strings.EqualFold(c.Name, fieldName) {
				col = &c
				break
			}
		}

		if col == nil {
			// If column not found in database, use the default value from template
			if defaultValue != nil {
				data[fieldName] = defaultValue
			} else {
				// Generate a default value based on field name
				value, err := g.generateValueForType("string", true, fieldName, ColumnInfo{})
				if err != nil {
					fmt.Printf("Warning: Failed to generate value for %s: %v\n", fieldName, err)
					continue
				}
				data[fieldName] = value
			}
			continue
		}

		// Skip auto-increment primary keys for POST requests
		if col.IsPrimary && col.IsAutoIncrement {
			continue
		}

		// Handle foreign key relationships
		if col.IsForeign {
			// Get a valid ID from the referenced table
			refValue, err := g.getValidForeignKeyValue(col.References, col.Name)
			if err != nil {
				fmt.Printf("Warning: Failed to get foreign key value for %s: %v\n", col.Name, err)
				continue
			}
			data[col.Name] = refValue
			continue
		}

		// If template has a default value, use it
		if defaultValue != nil && defaultValue != "" {
			data[fieldName] = defaultValue
			continue
		}

		// Generate value based on column type and name
		value, err := g.generateValueForType(col.Type, col.Nullable, col.Name, *col)
		if err != nil {
			fmt.Printf("Warning: Failed to generate value for %s: %v\n", col.Name, err)
			continue
		}

		// Apply max length constraint for string types
		if strValue, ok := value.(string); ok && col.MaxLength > 0 {
			if len(strValue) > col.MaxLength {
				value = strValue[:col.MaxLength]
			}
		}

		// Add to data map
		data[fieldName] = value
	}

	return data, nil
}

// generateValueForType generates a value based on the column type and constraints
func (g *DBGenerator) generateValueForType(colType string, nullable bool, columnName string, col ColumnInfo) (interface{}, error) {
	// Only return nil if the field is explicitly nullable and has a high chance
	if nullable && rand.Float32() < 0.1 { // Reduced chance of null from 0.2 to 0.1
		return nil, nil
	}

	// Generate value based on column name first (for common patterns)
	columnName = strings.ToLower(columnName)
	switch {
	case strings.Contains(columnName, "email"):
		return fmt.Sprintf("user_%d@example.com", rand.Intn(1000)), nil
	case strings.Contains(columnName, "phone"):
		return fmt.Sprintf("+1-%d-%d-%d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(9000)+1000), nil
	case strings.Contains(columnName, "first_name"):
		return fmt.Sprintf("John%d", rand.Intn(100)), nil
	case strings.Contains(columnName, "last_name"):
		return fmt.Sprintf("Doe%d", rand.Intn(100)), nil
	case strings.Contains(columnName, "address"):
		return fmt.Sprintf("%d Main St", rand.Intn(1000)+1), nil
	case strings.Contains(columnName, "city"):
		return fmt.Sprintf("City%d", rand.Intn(100)), nil
	case strings.Contains(columnName, "country"):
		return fmt.Sprintf("Country%d", rand.Intn(100)), nil
	case strings.Contains(columnName, "postal_code"), strings.Contains(columnName, "zip"):
		return fmt.Sprintf("%d%d", rand.Intn(90000)+10000, rand.Intn(1000)+100), nil
	case strings.Contains(columnName, "date_of_birth"):
		// Generate a date between 18 and 80 years ago
		years := rand.Intn(62) + 18
		return time.Now().AddDate(-years, 0, 0).Format("2006-01-02"), nil
	case strings.Contains(columnName, "username"):
		return fmt.Sprintf("user_%d", rand.Intn(1000)), nil
	case strings.Contains(columnName, "vat"):
		return fmt.Sprintf("VAT%d", rand.Intn(1000000)), nil
	case strings.Contains(columnName, "system_name"):
		return fmt.Sprintf("system_%d", rand.Intn(1000)), nil
	case strings.Contains(columnName, "timezone"):
		return "UTC", nil
	case strings.Contains(columnName, "gender"):
		genders := []string{"M", "F", "O"}
		return genders[rand.Intn(len(genders))], nil
	case strings.Contains(columnName, "company"):
		return fmt.Sprintf("Company%d", rand.Intn(1000)), nil
	case strings.Contains(columnName, "county"):
		return fmt.Sprintf("County%d", rand.Intn(100)), nil
	case strings.Contains(columnName, "comment"):
		return fmt.Sprintf("value_%d", rand.Intn(1000)), nil
	case strings.Contains(columnName, "guid"):
		return uuid.New().String(), nil
	case strings.Contains(columnName, "id"):
		return rand.Intn(1000) + 1, nil
	case strings.Contains(columnName, "created") || strings.Contains(columnName, "updated"):
		return time.Now().Format(time.RFC3339), nil
	case strings.Contains(columnName, "deleted"):
		return false, nil
	case strings.Contains(columnName, "active"):
		return true, nil
	}

	// If no specific pattern found, generate based on type
	switch strings.ToLower(colType) {
	case "integer", "int", "int4", "bigint", "int8":
		return rand.Intn(1000) + 1, nil
	case "numeric", "decimal", "real", "double precision", "float", "float4", "float8":
		return rand.Float64() * 1000, nil
	case "boolean", "bool":
		return rand.Float32() < 0.7, nil
	case "character varying", "varchar", "text", "char", "character":
		length := col.MaxLength
		if length == 0 {
			length = 10
		}
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rand.Intn(len(charset))]
		}
		return string(b), nil
	case "timestamp", "timestamp with time zone", "timestamptz", "timestamp without time zone":
		return time.Now().Add(time.Duration(rand.Intn(1000)) * time.Hour).Format(time.RFC3339), nil
	case "date":
		return time.Now().AddDate(0, 0, rand.Intn(365)).Format("2006-01-02"), nil
	case "time", "time with time zone", "timetz":
		return time.Now().Add(time.Duration(rand.Intn(24)) * time.Hour).Format("15:04:05"), nil
	case "uuid":
		return uuid.New().String(), nil
	case "user-defined":
		// For user-defined types, try to generate a reasonable value based on the column name
		if strings.Contains(columnName, "date") || strings.Contains(columnName, "time") {
			return time.Now().Format(time.RFC3339), nil
		}
		if strings.Contains(columnName, "name") {
			return fmt.Sprintf("Name%d", rand.Intn(1000)), nil
		}
		if strings.Contains(columnName, "code") {
			return fmt.Sprintf("CODE%d", rand.Intn(1000)), nil
		}
		if strings.Contains(columnName, "id") {
			return rand.Intn(1000) + 1, nil
		}
		// Default for user-defined types
		return fmt.Sprintf("value_%d", rand.Intn(1000)), nil
	default:
		// For unknown types, try to generate a reasonable value
		if strings.Contains(strings.ToLower(colType), "char") || strings.Contains(strings.ToLower(colType), "text") {
			return fmt.Sprintf("text_%d", rand.Intn(1000)), nil
		}
		if strings.Contains(strings.ToLower(colType), "int") || strings.Contains(strings.ToLower(colType), "number") {
			return rand.Intn(1000), nil
		}
		if strings.Contains(strings.ToLower(colType), "date") || strings.Contains(strings.ToLower(colType), "time") {
			return time.Now().Format(time.RFC3339), nil
		}
		return fmt.Sprintf("value_%d", rand.Intn(1000)), nil
	}
}

// getValidForeignKeyValue gets a valid ID from the referenced table
func (g *DBGenerator) getValidForeignKeyValue(refTable, columnName string) (interface{}, error) {
	// First check if the table exists
	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE LOWER(table_name) = LOWER($1)
		)
	`
	var exists bool
	err := g.db.QueryRow(checkQuery, refTable).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check if table exists: %v", err)
	}
	if !exists {
		if g.llmClient == nil {
			return nil, fmt.Errorf("referenced table '%s' not found and LLM client is not available", refTable)
		}

		fmt.Printf("Referenced table '%s' not found. Using LLM to suggest alternatives...\n", refTable)

		// Get schema information for LLM analysis
		schemaInfo := g.getSchemaInfo()

		// Use LLM to analyze relationships and suggest similar tables
		analysis, err := g.llmClient.AnalyzeRelationships(context.Background(), refTable, schemaInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze relationships with LLM: %v", err)
		}

		// Present suggestions to user
		fmt.Printf("\nSimilar tables found:\n")
		for i, suggestion := range analysis.Suggestions {
			fmt.Printf("%d. %s (Similarity: %.2f)\n", i+1, suggestion.TableName, suggestion.SimilarityScore)
			fmt.Printf("   Reason: %s\n", suggestion.Reasoning)
		}
		fmt.Printf("0. Enter custom table name\n")

		// Get user input
		var choice int
		fmt.Print("\nSelect a table (enter number): ")
		fmt.Scanln(&choice)

		if choice == 0 {
			// Get custom table name
			fmt.Print("Enter custom table name: ")
			fmt.Scanln(&refTable)
		} else if choice > 0 && choice <= len(analysis.Suggestions) {
			refTable = analysis.Suggestions[choice-1].TableName
		} else {
			return nil, fmt.Errorf("invalid selection")
		}
	}

	// Query to get a random valid ID from the referenced table
	// Quote both table name and column name to handle case sensitivity
	query := fmt.Sprintf(`SELECT "%s" FROM "%s" ORDER BY RANDOM() LIMIT 1`, columnName, refTable)
	var value interface{}
	err = g.db.QueryRow(query).Scan(&value)
	if err != nil {
		if g.llmClient == nil {
			return nil, fmt.Errorf("failed to get value from table '%s' and LLM client is not available", refTable)
		}

		fmt.Printf("Failed to get value from table '%s'. Using LLM to suggest value...\n", refTable)

		// Use LLM to analyze the column and suggest a value
		analysis, err := g.llmClient.AnalyzeColumn(context.Background(), refTable, columnName, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze column with LLM: %v", err)
		}

		// Present suggestions to user
		fmt.Printf("\nSuggested value types for '%s.%s':\n", refTable, columnName)
		fmt.Printf("1. %s\n", analysis.DataPatterns.DataType)
		if len(analysis.DataPatterns.ValueRange) > 0 {
			fmt.Printf("2. Use one of these values: %v\n", analysis.DataPatterns.ValueRange)
		}
		fmt.Printf("3. Enter custom value\n")

		// Get user input
		var choice int
		fmt.Print("\nSelect an option (enter number): ")
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			// Generate value based on suggested type
			value, err = g.generateValueForType(analysis.DataPatterns.DataType, true, columnName, ColumnInfo{})
		case 2:
			if len(analysis.DataPatterns.ValueRange) > 0 {
				// Use a random value from the range
				value = analysis.DataPatterns.ValueRange[rand.Intn(len(analysis.DataPatterns.ValueRange))]
			} else {
				value, err = g.generateValueForType(analysis.DataPatterns.DataType, true, columnName, ColumnInfo{})
			}
		case 3:
			// Get custom value
			fmt.Print("Enter custom value: ")
			var customValue string
			fmt.Scanln(&customValue)
			value = customValue
		default:
			return nil, fmt.Errorf("invalid selection")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to generate value: %v", err)
		}
	}

	return value, nil
}
