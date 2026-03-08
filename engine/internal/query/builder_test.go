package query

import (
	"strings"
	"testing"

	"github.com/practor/practor-engine/internal/schema"
)

func TestBuildCreateTableEscapesDefaultsAndMappedForeignKeys(t *testing.T) {
	s := &schema.Schema{
		Enums: []schema.Enum{
			{
				Name:   "Role",
				DBName: `role"type`,
				Values: []schema.EnumValue{
					{Name: "USER"},
					{Name: "ADMIN", DBName: "admin'value"},
				},
			},
		},
		Models: []schema.Model{
			{
				Name:   "User",
				DBName: `app"users`,
				Fields: []schema.Field{
					{
						Name: "id",
						Type: schema.FieldType{IsScalar: true, Name: "Int"},
						Attributes: []schema.FieldAttribute{
							{Name: "id"},
							{Name: "map", Args: map[string]interface{}{"_0": `user"id`}},
						},
					},
				},
			},
			{
				Name: "Post",
				Fields: []schema.Field{
					{
						Name: "title",
						Type: schema.FieldType{IsScalar: true, Name: "String"},
						DefaultValue: &schema.DefaultValue{
							Type:  schema.DefaultValueLiteral,
							Value: "O'Reilly",
						},
					},
					{
						Name: "role",
						Type: schema.FieldType{Name: "Role", IsEnum: true},
						DefaultValue: &schema.DefaultValue{
							Type:  schema.DefaultValueEnum,
							Value: "ADMIN",
						},
					},
					{
						Name: "authorId",
						Type: schema.FieldType{IsScalar: true, Name: "Int"},
						Attributes: []schema.FieldAttribute{
							{Name: "map", Args: map[string]interface{}{"_0": "author_id"}},
						},
					},
					{
						Name: "author",
						Type: schema.FieldType{Name: "User", IsModel: true},
						Attributes: []schema.FieldAttribute{
							{
								Name: "relation",
								Args: map[string]interface{}{
									"fields":     []interface{}{"authorId"},
									"references": []interface{}{"id"},
								},
							},
						},
					},
				},
			},
		},
	}

	builder := NewBuilder("postgresql", s)
	postSQL := builder.BuildCreateTable(&s.Models[1])
	enumSQL := builder.BuildCreateEnum(&s.Enums[0])

	if !strings.Contains(postSQL, `DEFAULT 'O''Reilly'`) {
		t.Fatalf("expected escaped string default, got:\n%s", postSQL)
	}

	if !strings.Contains(postSQL, `DEFAULT 'admin''value'`) {
		t.Fatalf("expected escaped mapped enum default, got:\n%s", postSQL)
	}

	if !strings.Contains(postSQL, `FOREIGN KEY ("author_id") REFERENCES "app""users" ("user""id")`) {
		t.Fatalf("expected mapped FK constraint, got:\n%s", postSQL)
	}

	if !strings.Contains(enumSQL, `CREATE TYPE "role""type" AS ENUM ('USER', 'admin''value')`) {
		t.Fatalf("expected escaped mapped enum DDL, got:\n%s", enumSQL)
	}
}

func TestCollectParentValuesUsesMappedColumnNames(t *testing.T) {
	s := &schema.Schema{
		Models: []schema.Model{
			{
				Name: "User",
				Fields: []schema.Field{
					{
						Name: "id",
						Type: schema.FieldType{IsScalar: true, Name: "Int"},
						Attributes: []schema.FieldAttribute{
							{Name: "map", Args: map[string]interface{}{"_0": "user_id"}},
						},
					},
				},
			},
		},
	}

	engine := &Engine{
		schema:  s,
		builder: NewBuilder("postgresql", s),
	}

	values, columnName := engine.collectParentValues("User", []map[string]interface{}{
		{"user_id": 7},
		{"user_id": 7},
		{"user_id": 9},
	}, "id")

	if columnName != "user_id" {
		t.Fatalf("expected mapped column name user_id, got %s", columnName)
	}

	if len(values) != 2 || values[0] != 7 || values[1] != 9 {
		t.Fatalf("unexpected collected values: %#v", values)
	}
}
