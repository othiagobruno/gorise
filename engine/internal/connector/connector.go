package connector

import (
	"context"
	"database/sql"
	"time"
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

// PoolConfig holds configurable connection pool parameters.
// Zero-values fall back to sensible defaults inside each connector.
type PoolConfig struct {
	// MaxOpenConns is the maximum number of open connections to the database.
	// Default: 20
	MaxOpenConns int `json:"maxOpenConns,omitempty"`

	// MaxIdleConns is the maximum number of idle connections in the pool.
	// Default: 5
	MaxIdleConns int `json:"maxIdleConns,omitempty"`

	// ConnMaxLifetimeMs is the maximum lifetime of a connection in milliseconds.
	// Default: 300000 (5 minutes)
	ConnMaxLifetimeMs int `json:"connMaxLifetimeMs,omitempty"`

	// ConnMaxIdleTimeMs is the maximum idle time of a connection in milliseconds.
	// Default: 60000 (1 minute)
	ConnMaxIdleTimeMs int `json:"connMaxIdleTimeMs,omitempty"`
}

// Apply sets pool parameters on *sql.DB using sensible defaults for zero-values.
func (pc PoolConfig) Apply(db *sql.DB) {
	maxOpen := pc.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 20
	}
	maxIdle := pc.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	maxLifetime := pc.ConnMaxLifetimeMs
	if maxLifetime <= 0 {
		maxLifetime = 300_000 // 5 minutes
	}
	maxIdleTime := pc.ConnMaxIdleTimeMs
	if maxIdleTime <= 0 {
		maxIdleTime = 60_000 // 1 minute
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Millisecond)
	db.SetConnMaxIdleTime(time.Duration(maxIdleTime) * time.Millisecond)
}

// PoolStats represents runtime statistics of the connection pool.
type PoolStats struct {
	MaxOpenConnections int `json:"maxOpenConnections"`
	OpenConnections    int `json:"openConnections"`
	InUse              int `json:"inUse"`
	Idle               int `json:"idle"`
	WaitCount          int `json:"waitCount"`
	WaitDurationMs     int `json:"waitDurationMs"`
	MaxIdleClosed      int `json:"maxIdleClosed"`
	MaxIdleTimeClosed  int `json:"maxIdleTimeClosed"`
	MaxLifetimeClosed  int `json:"maxLifetimeClosed"`
}

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

	// GetPoolStats returns runtime statistics of the connection pool.
	GetPoolStats() *PoolStats
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
