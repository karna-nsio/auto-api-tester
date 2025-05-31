package generator

import (
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
}

// NewDBGenerator creates a new instance of DBGenerator
func NewDBGenerator(config DBConfig, templatePath, outputPath string) *DBGenerator {
	// Initialize random number generator
	rand.Seed(time.Now().UnixNano())

	return &DBGenerator{
		config:       config,
		templatePath: templatePath,
		outputPath:   outputPath,
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

	// Generate data based on HTTP method and database tables
	switch method {
	case "GET":
		return g.generateGetData(path, testData, tables)
	case "POST":
		return g.generatePostData(path, testData, tables)
	case "PUT":
		return g.generatePutData(path, testData, tables)
	case "DELETE":
		return g.generateDeleteData(path, testData, tables)
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

	// Find related tables
	relatedTables, err := g.analyzer.FindRelatedTables(tableName)
	if err != nil {
		return nil, err
	}

	// Add the main table to the list
	tables := append([]string{tableName}, relatedTables...)
	return tables, nil
}

// generateGetData generates test data for GET endpoints
func (g *DBGenerator) generateGetData(path string, data types.EndpointTestData, tables []string) (types.EndpointTestData, error) {
	// Generate query parameters
	if len(data.QueryParams) > 0 {
		for param, value := range data.QueryParams {
			if value == nil {
				// Generate value from database
				generatedValue, err := g.generateValueFromDB(param, tables)
				if err != nil {
					return data, err
				}
				data.QueryParams[param] = generatedValue
			}
		}
	}

	// Generate path parameters
	if len(data.PathParams) > 0 {
		for param, value := range data.PathParams {
			if value == nil {
				// Generate value from database
				generatedValue, err := g.generateValueFromDB(param, tables)
				if err != nil {
					return data, err
				}
				data.PathParams[param] = generatedValue
			}
		}
	}

	return data, nil
}

// generatePostData generates test data for POST endpoints
func (g *DBGenerator) generatePostData(path string, data types.EndpointTestData, tables []string) (types.EndpointTestData, error) {
	// Generate request body
	// if data.Body == nil {
	// Generate body data from database tables
	generatedBody, err := g.generateBodyFromDB(tables)
	if err != nil {
		return data, err
	}
	data.Body = generatedBody
	// }

	return data, nil
}

// generatePutData generates test data for PUT endpoints
func (g *DBGenerator) generatePutData(path string, data types.EndpointTestData, tables []string) (types.EndpointTestData, error) {
	// Similar to POST, but we need to ensure we have an ID
	return g.generatePostData(path, data, tables)
}

// generateDeleteData generates test data for DELETE endpoints
func (g *DBGenerator) generateDeleteData(path string, data types.EndpointTestData, tables []string) (types.EndpointTestData, error) {
	// Similar to GET, but we only need the ID
	return g.generateGetData(path, data, tables)
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

	// If no matching column found, return a default value
	return nil, nil
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

	// Create a map to hold the generated data
	data := make(map[string]interface{})

	// Generate values for each column
	fmt.Println("tableInfo", tableInfo)
	for _, col := range tableInfo.Columns {
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

		// Generate value based on column type and name
		value, err := g.generateValueForType(col.Type, col.Nullable, col.Name, col)
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
		data[col.Name] = value
	}

	return data, nil
}

// generateValueForType generates a value based on the column type and constraints
func (g *DBGenerator) generateValueForType(colType string, nullable bool, columnName string, col ColumnInfo) (interface{}, error) {
	// If nullable and random chance, return nil
	if nullable && rand.Float32() < 0.2 {
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
		// If table doesn't exist, generate a random ID
		return rand.Intn(1000) + 1, nil
	}

	// Query to get a random valid ID from the referenced table
	query := fmt.Sprintf("SELECT %s FROM %s ORDER BY RANDOM() LIMIT 1", columnName, refTable)
	var value interface{}
	err = g.db.QueryRow(query).Scan(&value)
	if err != nil {
		// If query fails, generate a random ID
		return rand.Intn(1000) + 1, nil
	}
	return value, nil
}
