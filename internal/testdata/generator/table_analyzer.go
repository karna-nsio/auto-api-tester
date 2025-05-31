package generator

import (
	"database/sql"
	"fmt"
	"strings"
)

// TableInfo represents information about a database table
type TableInfo struct {
	Name        string
	Columns     []ColumnInfo
	PrimaryKey  string
	ForeignKeys []ForeignKeyInfo
}

// ColumnInfo represents information about a database column
type ColumnInfo struct {
	Name            string
	Type            string
	Nullable        bool
	IsPrimary       bool
	IsForeign       bool
	References      string
	Default         interface{}
	MaxLength       int
	IsUnique        bool
	IsAutoIncrement bool
	CheckConstraint string
	EnumValues      []string
	Precision       int
	Scale           int
	MinValue        interface{}
	MaxValue        interface{}
	Pattern         string
	DomainName      string
	Comment         string
}

// ForeignKeyInfo represents information about a foreign key relationship
type ForeignKeyInfo struct {
	Column           string
	ReferencedTable  string
	ReferencedColumn string
}

// TableAnalyzer handles database schema analysis
type TableAnalyzer struct {
	db *sql.DB
}

// NewTableAnalyzer creates a new instance of TableAnalyzer
func NewTableAnalyzer(db *sql.DB) *TableAnalyzer {
	return &TableAnalyzer{db: db}
}

// AnalyzeTables analyzes all tables in the database
func (ta *TableAnalyzer) AnalyzeTables() (map[string]TableInfo, error) {
	tables := make(map[string]TableInfo)

	// Get list of tables
	tableNames, err := ta.getTableNames()
	if err != nil {
		return nil, err
	}

	// Analyze each table
	for _, tableName := range tableNames {
		tableInfo, err := ta.analyzeTable(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze table %s: %v", tableName, err)
		}
		tables[tableName] = tableInfo
	}

	return tables, nil
}

// getTableNames retrieves all table names from the database
func (ta *TableAnalyzer) getTableNames() ([]string, error) {
	var tables []string
	query := `
		SELECT LOWER(table_name) 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
	`
	rows, err := ta.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// analyzeTable analyzes a single table's structure
func (ta *TableAnalyzer) analyzeTable(tableName string) (TableInfo, error) {
	info := TableInfo{
		Name: tableName,
	}

	// Get column information
	columns, err := ta.getColumnInfo(tableName)
	if err != nil {
		return info, err
	}
	info.Columns = columns

	// Get primary key
	pk, err := ta.getPrimaryKey(tableName)
	if err != nil {
		return info, err
	}
	info.PrimaryKey = pk

	// Get foreign keys
	fks, err := ta.getForeignKeys(tableName)
	if err != nil {
		return info, err
	}
	info.ForeignKeys = fks

	return info, nil
}

// getColumnInfo retrieves column information for a table
func (ta *TableAnalyzer) getColumnInfo(tableName string) ([]ColumnInfo, error) {
	var columns []ColumnInfo
	query := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale
		FROM information_schema.columns c
		WHERE LOWER(c.table_name) = LOWER($1)
		ORDER BY c.column_name
	`
	rows, err := ta.db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var nullable string
		var maxLength sql.NullInt64
		var precision, scale sql.NullInt64

		if err := rows.Scan(
			&col.Name,
			&col.Type,
			&nullable,
			&col.Default,
			&maxLength,
			&precision,
			&scale,
		); err != nil {
			return nil, err
		}

		col.Nullable = nullable == "YES"
		if maxLength.Valid {
			col.MaxLength = int(maxLength.Int64)
		}
		if precision.Valid {
			col.Precision = int(precision.Int64)
		}
		if scale.Valid {
			col.Scale = int(scale.Int64)
		}

		columns = append(columns, col)
	}

	// Get primary key information
	pkQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
		AND LOWER(tc.table_name) = LOWER($1)
	`
	rows, err = ta.db.Query(pkQuery, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pkColumn string
		if err := rows.Scan(&pkColumn); err != nil {
			return nil, err
		}
		// Mark column as primary key
		for i := range columns {
			if columns[i].Name == pkColumn {
				columns[i].IsPrimary = true
				break
			}
		}
	}

	// Get foreign key information
	fkQuery := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND LOWER(tc.table_name) = LOWER($1)
	`
	rows, err = ta.db.Query(fkQuery, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fkColumn, refTable, refColumn string
		if err := rows.Scan(&fkColumn, &refTable, &refColumn); err != nil {
			return nil, err
		}
		// Mark column as foreign key
		for i := range columns {
			if columns[i].Name == fkColumn {
				columns[i].IsForeign = true
				columns[i].References = refTable
				break
			}
		}
	}

	return columns, nil
}

// parseCheckConstraint extracts min/max values from check constraints
func parseCheckConstraint(constraint string) (min, max interface{}) {
	constraint = strings.ToLower(constraint)

	// Handle range constraints
	if strings.Contains(constraint, "between") {
		var minVal, maxVal float64
		fmt.Sscanf(constraint, "check (%s between %f and %f)", &minVal, &maxVal)
		return minVal, maxVal
	}

	// Handle >= and <= constraints
	if strings.Contains(constraint, ">=") {
		var minVal float64
		fmt.Sscanf(constraint, "check (%s >= %f)", &minVal)
		return minVal, nil
	}
	if strings.Contains(constraint, "<=") {
		var maxVal float64
		fmt.Sscanf(constraint, "check (%s <= %f)", &maxVal)
		return nil, maxVal
	}

	// Handle pattern constraints
	if strings.Contains(constraint, "like") {
		pattern := strings.Trim(constraint, "'")
		return nil, pattern
	}

	return nil, nil
}

// getPrimaryKey retrieves the primary key for a table
func (ta *TableAnalyzer) getPrimaryKey(tableName string) (string, error) {
	query := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
		AND LOWER(tc.table_name) = LOWER($1)
	`
	var pk string
	err := ta.db.QueryRow(query, tableName).Scan(&pk)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return pk, nil
}

// getForeignKeys retrieves foreign key information for a table
func (ta *TableAnalyzer) getForeignKeys(tableName string) ([]ForeignKeyInfo, error) {
	var fks []ForeignKeyInfo
	query := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND LOWER(tc.table_name) = LOWER($1)
	`
	rows, err := ta.db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fk ForeignKeyInfo
		var updateRule, deleteRule string
		if err := rows.Scan(
			&fk.Column,
			&fk.ReferencedTable,
			&fk.ReferencedColumn,
			&updateRule,
			&deleteRule,
		); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}

	return fks, nil
}

// FindRelatedTables finds tables related to a given table through foreign keys
func (ta *TableAnalyzer) FindRelatedTables(tableName string) ([]string, error) {
	var relatedTables []string
	query := `
		SELECT DISTINCT ccu.table_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND (tc.table_name = $1 OR ccu.table_name = $1)
	`
	rows, err := ta.db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var relatedTable string
		if err := rows.Scan(&relatedTable); err != nil {
			return nil, err
		}
		if relatedTable != tableName {
			relatedTables = append(relatedTables, relatedTable)
		}
	}

	return relatedTables, nil
}
