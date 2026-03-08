package connector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// ============================================================================
// PostgreSQL Connector
// ============================================================================

// PostgresConnector implements the Connector interface for PostgreSQL.
type PostgresConnector struct {
	db   *sql.DB
	dsn  string
	pool PoolConfig
}

// NewPostgresConnector creates a new PostgreSQL connector with pool configuration.
func NewPostgresConnector(dsn string, pool PoolConfig) *PostgresConnector {
	return &PostgresConnector{dsn: dsn, pool: pool}
}

// Connect establishes a connection to the PostgreSQL database.
func (c *PostgresConnector) Connect(ctx context.Context) error {
	db, err := sql.Open("postgres", c.dsn)
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Apply configurable pool settings (with sensible defaults)
	c.pool.Apply(db)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	c.db = db
	return nil
}

// Disconnect closes the database connection.
func (c *PostgresConnector) Disconnect(ctx context.Context) error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Execute runs a non-returning query.
func (c *PostgresConnector) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

// Query runs a query that returns rows.
func (c *PostgresConnector) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// QueryRow runs a query that returns at most one row.
func (c *PostgresConnector) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// GetDialect returns the PostgreSQL dialect.
func (c *PostgresConnector) GetDialect() Dialect {
	return DialectPostgres
}

// GetDB returns the underlying *sql.DB.
func (c *PostgresConnector) GetDB() *sql.DB {
	return c.db
}

// Ping verifies the connection.
func (c *PostgresConnector) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// BeginTx starts a new transaction.
func (c *PostgresConnector) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return c.db.BeginTx(ctx, opts)
}

// GetPoolStats returns runtime connection pool statistics from the underlying *sql.DB.
func (c *PostgresConnector) GetPoolStats() *PoolStats {
	if c.db == nil {
		return &PoolStats{}
	}
	s := c.db.Stats()
	return &PoolStats{
		MaxOpenConnections: s.MaxOpenConnections,
		OpenConnections:    s.OpenConnections,
		InUse:              s.InUse,
		Idle:               s.Idle,
		WaitCount:          int(s.WaitCount),
		WaitDurationMs:     int(s.WaitDuration.Milliseconds()),
		MaxIdleClosed:      int(s.MaxIdleClosed),
		MaxIdleTimeClosed:  int(s.MaxIdleTimeClosed),
		MaxLifetimeClosed:  int(s.MaxLifetimeClosed),
	}
}

// ============================================================================
// PostgreSQL-specific SQL helpers
// ============================================================================

// PostgresDialect provides PostgreSQL-specific SQL generation.
type PostgresDialect struct{}

// Placeholder returns the PostgreSQL placeholder format ($1, $2, ...).
func (d *PostgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// QuoteIdentifier quotes a PostgreSQL identifier.
func (d *PostgresDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

// CreateTableSQL generates CREATE TABLE SQL for PostgreSQL.
func (d *PostgresDialect) AutoIncrementType() string {
	return "SERIAL"
}

// BigAutoIncrementType returns the PostgreSQL big auto-increment type.
func (d *PostgresDialect) BigAutoIncrementType() string {
	return "BIGSERIAL"
}

// TypeMapping maps Practor scalar types to PostgreSQL types.
func (d *PostgresDialect) TypeMapping() map[string]string {
	return map[string]string{
		"String":   "TEXT",
		"Int":      "INTEGER",
		"Float":    "DOUBLE PRECISION",
		"Boolean":  "BOOLEAN",
		"DateTime": "TIMESTAMP(3)",
		"Json":     "JSONB",
		"BigInt":   "BIGINT",
		"Bytes":    "BYTEA",
		"Decimal":  "DECIMAL(65,30)",
	}
}
