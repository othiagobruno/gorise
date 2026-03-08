package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/practor/practor-engine/internal/connector"
	"github.com/practor/practor-engine/internal/protocol"
	"github.com/practor/practor-engine/internal/schema"
)

// Version is set at build time.
var Version = "0.3.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("practor-engine v%s\n", Version)
			os.Exit(0)

		case "parse":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: practor parse <schema-file>")
				os.Exit(1)
			}
			handleParse(os.Args[2])
			os.Exit(0)

		case "validate":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: practor validate <schema-file>")
				os.Exit(1)
			}
			handleValidate(os.Args[2])
			os.Exit(0)
		}
	}

	// Default: start JSON-RPC server mode
	startServer()
}

// envInt reads an integer from an environment variable, returning 0 if unset or invalid.
func envInt(key string) int {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

func startServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Read the DATABASE_URL from environment
	dsn := os.Getenv("DATABASE_URL")

	// Read schema path from env or default
	schemaPath := os.Getenv("PRACTOR_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = "schema.practor"
	}

	// Parse schema
	var parsedSchema *schema.Schema
	if data, err := os.ReadFile(schemaPath); err == nil {
		parsed, err := schema.Parse(string(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Schema parse error: %v\n", err)
			os.Exit(1)
		}
		schema.ResolveFieldTypes(parsed)
		parsedSchema = parsed

		// Extract DSN from datasource if not in env
		if dsn == "" && len(parsed.Datasources) > 0 {
			ds := parsed.Datasources[0]
			if ds.IsEnvVar {
				dsn = os.Getenv(ds.EnvVarName)
			} else {
				dsn = ds.URL
			}
		}
	} else {
		// No schema file — create empty schema
		parsedSchema = &schema.Schema{}
	}

	// Build connection pool configuration from PRACTOR_POOL_* env vars.
	// Zero-values are fine — PoolConfig.Apply() uses sensible defaults.
	poolCfg := connector.PoolConfig{
		MaxOpenConns:      envInt("PRACTOR_POOL_MAX_OPEN_CONNS"),
		MaxIdleConns:      envInt("PRACTOR_POOL_MAX_IDLE_CONNS"),
		ConnMaxLifetimeMs: envInt("PRACTOR_POOL_CONN_MAX_LIFETIME_MS"),
		ConnMaxIdleTimeMs: envInt("PRACTOR_POOL_CONN_MAX_IDLE_TIME_MS"),
	}

	// Create connector
	var conn connector.Connector
	if dsn != "" {
		pgConn := connector.NewPostgresConnector(dsn, poolCfg)
		if err := pgConn.Connect(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Database connection error: %v\n", err)
			// Don't exit — engine can still parse schemas without DB
		} else {
			conn = pgConn
			defer pgConn.Disconnect(ctx)
		}
	}

	// Create and start JSON-RPC server
	server := protocol.NewServer()

	if conn != nil {
		protocol.NewEngineHandler(server, conn, parsedSchema)
	} else {
		// Register minimal handlers when no DB connection
		server.RegisterHandler("schema.parse", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			var sp protocol.SchemaParams
			if err := json.Unmarshal(params, &sp); err != nil {
				return nil, err
			}
			input := sp.Schema
			if input == "" && sp.SchemaPath != "" {
				data, err := os.ReadFile(sp.SchemaPath)
				if err != nil {
					return nil, err
				}
				input = string(data)
			}
			parsed, err := schema.Parse(input)
			if err != nil {
				return &protocol.SchemaResponse{Valid: false, Errors: []string{err.Error()}}, nil
			}
			schema.ResolveFieldTypes(parsed)
			return &protocol.SchemaResponse{Schema: parsed, Valid: true}, nil
		})

		server.RegisterHandler("schema.getJSON", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return parsedSchema, nil
		})

		server.RegisterHandler("ping", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return map[string]string{"status": "pong", "db": "disconnected"}, nil
		})
	}

	// Start the server (blocks until stdin closes or signal received)
	if err := server.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func handleParse(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	parsed, err := schema.Parse(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	schema.ResolveFieldTypes(parsed)

	jsonData, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

func handleValidate(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	parsed, err := schema.Parse(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	schema.ResolveFieldTypes(parsed)
	errors := schema.Validate(parsed)

	if len(errors) > 0 {
		fmt.Fprintln(os.Stderr, "Validation errors:")
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  ❌ %s\n", e.Error())
		}
		os.Exit(1)
	}

	fmt.Println("✅ Schema is valid")
}
