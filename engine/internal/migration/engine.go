package migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/practor/practor-engine/internal/connector"
	"github.com/practor/practor-engine/internal/query"
	"github.com/practor/practor-engine/internal/schema"
)

// ============================================================================
// Migration Engine — Manages database schema migrations
// ============================================================================

// Engine manages migrations.
type Engine struct {
	connector    connector.Connector
	queryBuilder *query.Builder
}

// NewEngine creates a new migration Engine.
func NewEngine(conn connector.Connector, s *schema.Schema) *Engine {
	return &Engine{
		connector:    conn,
		queryBuilder: query.NewBuilder(string(conn.GetDialect()), s),
	}
}

// Migration represents a single migration.
type Migration struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SQL       string    `json:"sql"`
	AppliedAt time.Time `json:"appliedAt"`
}

// MigrationStatus represents the current migration status.
type MigrationStatus struct {
	Applied []Migration `json:"applied"`
	Pending []Migration `json:"pending"`
}

// ============================================================================
// Migration tracking table
// ============================================================================

const migrationsTableSQL = `
CREATE TABLE IF NOT EXISTS "_practor_migrations" (
  "id" TEXT PRIMARY KEY,
  "name" TEXT NOT NULL,
  "applied_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "sql_content" TEXT NOT NULL
)
`

// EnsureMigrationsTable creates the migrations tracking table if it doesn't exist.
func (e *Engine) EnsureMigrationsTable(ctx context.Context) error {
	_, err := e.connector.Execute(ctx, migrationsTableSQL)
	return err
}

// ============================================================================
// Schema diffing
// ============================================================================

// Diff represents a schema difference.
type Diff struct {
	Type    DiffType `json:"type"`
	Model   string   `json:"model,omitempty"`
	Field   string   `json:"field,omitempty"`
	Details string   `json:"details"`
	SQL     string   `json:"sql"`
}

// DiffType represents the type of schema change.
type DiffType string

const (
	DiffCreateModel DiffType = "CREATE_MODEL"
	DiffDropModel   DiffType = "DROP_MODEL"
	DiffAddField    DiffType = "ADD_FIELD"
	DiffDropField   DiffType = "DROP_FIELD"
	DiffAlterField  DiffType = "ALTER_FIELD"
	DiffCreateEnum  DiffType = "CREATE_ENUM"
	DiffDropEnum    DiffType = "DROP_ENUM"
	DiffAddIndex    DiffType = "ADD_INDEX"
	DiffDropIndex   DiffType = "DROP_INDEX"
)

// DiffSchemas compares two schemas and returns the differences.
func DiffSchemas(from, to *schema.Schema) []Diff {
	var diffs []Diff

	fromModels := make(map[string]*schema.Model)
	toModels := make(map[string]*schema.Model)

	if from != nil {
		for i := range from.Models {
			fromModels[from.Models[i].Name] = &from.Models[i]
		}
	}
	for i := range to.Models {
		toModels[to.Models[i].Name] = &to.Models[i]
	}

	// Find new models
	for name, model := range toModels {
		if _, exists := fromModels[name]; !exists {
			diffs = append(diffs, Diff{
				Type:    DiffCreateModel,
				Model:   name,
				Details: fmt.Sprintf("Create model '%s' with %d fields", name, len(model.Fields)),
			})
		}
	}

	// Find dropped models
	if from != nil {
		for name := range fromModels {
			if _, exists := toModels[name]; !exists {
				diffs = append(diffs, Diff{
					Type:    DiffDropModel,
					Model:   name,
					Details: fmt.Sprintf("Drop model '%s'", name),
				})
			}
		}
	}

	// Find field changes in existing models
	for name, toModel := range toModels {
		fromModel, exists := fromModels[name]
		if !exists {
			continue
		}

		fromFields := make(map[string]*schema.Field)
		toFields := make(map[string]*schema.Field)

		for i := range fromModel.Fields {
			fromFields[fromModel.Fields[i].Name] = &fromModel.Fields[i]
		}
		for i := range toModel.Fields {
			toFields[toModel.Fields[i].Name] = &toModel.Fields[i]
		}

		// New fields
		for fieldName, field := range toFields {
			if _, exists := fromFields[fieldName]; !exists {
				diffs = append(diffs, Diff{
					Type:    DiffAddField,
					Model:   name,
					Field:   fieldName,
					Details: fmt.Sprintf("Add field '%s' (%s) to model '%s'", fieldName, field.Type.Name, name),
				})
			}
		}

		// Dropped fields
		for fieldName := range fromFields {
			if _, exists := toFields[fieldName]; !exists {
				diffs = append(diffs, Diff{
					Type:    DiffDropField,
					Model:   name,
					Field:   fieldName,
					Details: fmt.Sprintf("Drop field '%s' from model '%s'", fieldName, name),
				})
			}
		}

		// Modified fields
		for fieldName, toField := range toFields {
			fromField, exists := fromFields[fieldName]
			if !exists {
				continue
			}

			if fromField.Type.Name != toField.Type.Name ||
				fromField.IsOptional != toField.IsOptional ||
				fromField.IsList != toField.IsList {
				diffs = append(diffs, Diff{
					Type:    DiffAlterField,
					Model:   name,
					Field:   fieldName,
					Details: fmt.Sprintf("Alter field '%s' in model '%s'", fieldName, name),
				})
			}
		}
	}

	// Enum changes
	fromEnums := make(map[string]*schema.Enum)
	toEnums := make(map[string]*schema.Enum)

	if from != nil {
		for i := range from.Enums {
			fromEnums[from.Enums[i].Name] = &from.Enums[i]
		}
	}
	for i := range to.Enums {
		toEnums[to.Enums[i].Name] = &to.Enums[i]
	}

	for name := range toEnums {
		if _, exists := fromEnums[name]; !exists {
			diffs = append(diffs, Diff{
				Type:    DiffCreateEnum,
				Model:   name,
				Details: fmt.Sprintf("Create enum '%s'", name),
			})
		}
	}

	if from != nil {
		for name := range fromEnums {
			if _, exists := toEnums[name]; !exists {
				diffs = append(diffs, Diff{
					Type:    DiffDropEnum,
					Model:   name,
					Details: fmt.Sprintf("Drop enum '%s'", name),
				})
			}
		}
	}

	return diffs
}

// GenerateMigrationSQL generates SQL for a list of diffs.
func GenerateMigrationSQL(diffs []Diff, s *schema.Schema, dialect string) string {
	builder := query.NewBuilder(dialect, s)
	var statements []string

	for _, diff := range diffs {
		switch diff.Type {
		case DiffCreateModel:
			for i := range s.Models {
				if s.Models[i].Name == diff.Model {
					statements = append(statements, builder.BuildCreateTable(&s.Models[i]))
				}
			}
		case DiffDropModel:
			tableName := toSnakeCase(diff.Model)
			statements = append(statements, fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE`, tableName))
		case DiffCreateEnum:
			for i := range s.Enums {
				if s.Enums[i].Name == diff.Model {
					statements = append(statements, builder.BuildCreateEnum(&s.Enums[i]))
				}
			}
		case DiffDropEnum:
			enumName := toSnakeCase(diff.Model)
			statements = append(statements, fmt.Sprintf(`DROP TYPE IF EXISTS "%s"`, enumName))
		case DiffAddField:
			// Will be implemented with ALTER TABLE support
		case DiffDropField:
			tableName := toSnakeCase(diff.Model)
			colName := toSnakeCase(diff.Field)
			statements = append(statements, fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN IF EXISTS "%s"`, tableName, colName))
		}
	}

	return strings.Join(statements, ";\n\n") + ";"
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
