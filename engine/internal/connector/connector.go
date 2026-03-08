package connector

import (
	"context"
	"database/sql"
)

// ============================================================================
// Connector — Database connector interface
// ============================================================================

// Dialect represents the SQL dialect for a database.
type Dialect string

const (
	DialectPostgres  Dialect = "postgresql"
	DialectMySQL     Dialect = "mysql"
	DialectSQLite    Dialect = "sqlite"
	DialectSQLServer Dialect = "sqlserver"
)

// Connector defines the interface for database connections.
type Connector interface {
	// Connect establishes a connection to the database.
	Connect(ctx context.Context) error

	// Disconnect closes the database connection.
	Disconnect(ctx context.Context) error

	// Execute runs a query that does not return rows (INSERT, UPDATE, DELETE).
	Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Query runs a query that returns rows (SELECT).
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow runs a query that returns at most one row.
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

	// GetDialect returns the SQL dialect for this connector.
	GetDialect() Dialect

	// GetDB returns the underlying *sql.DB for advanced operations.
	GetDB() *sql.DB

	// Ping verifies the database connection.
	Ping(ctx context.Context) error

	// BeginTx starts a new transaction.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// QueryResult represents the result of a database query as a list of rows.
type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Count   int64                    `json:"count"`
}

// ScanRows scans all rows from a *sql.Rows into a QueryResult.
func ScanRows(rows *sql.Rows) (*QueryResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert byte slices to strings for JSON compatibility
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result.Rows = append(result.Rows, row)
		result.Count++
	}

	return result, rows.Err()
}
