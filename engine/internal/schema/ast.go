package schema

// ============================================================================
// AST — Abstract Syntax Tree for Practor Schema Language (PSL-compatible)
// ============================================================================

// Schema represents the entire parsed schema file.
type Schema struct {
	Datasources []Datasource `json:"datasources"`
	Generators  []Generator  `json:"generators"`
	Models      []Model      `json:"models"`
	Enums       []Enum       `json:"enums"`
}

// Datasource represents a `datasource` block.
type Datasource struct {
	Name       string            `json:"name"`
	Provider   string            `json:"provider"`
	URL        string            `json:"url"`
	IsEnvVar   bool              `json:"isEnvVar"`
	EnvVarName string            `json:"envVarName,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Generator represents a `generator` block.
type Generator struct {
	Name       string            `json:"name"`
	Provider   string            `json:"provider"`
	Output     string            `json:"output,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Model represents a `model` block with fields and attributes.
type Model struct {
	Name       string           `json:"name"`
	Fields     []Field          `json:"fields"`
	Attributes []ModelAttribute `json:"attributes,omitempty"`
	DBName     string           `json:"dbName,omitempty"` // @@map("table_name")
}

// Field represents a single field within a model.
type Field struct {
	Name         string           `json:"name"`
	Type         FieldType        `json:"type"`
	IsList       bool             `json:"isList"`
	IsOptional   bool             `json:"isOptional"`
	Attributes   []FieldAttribute `json:"attributes,omitempty"`
	DefaultValue *DefaultValue    `json:"defaultValue,omitempty"`
	Documentation string          `json:"documentation,omitempty"`
}

// FieldType represents the type of a field.
type FieldType struct {
	Name     string   `json:"name"`     // Scalar type name or model/enum reference
	IsScalar bool     `json:"isScalar"` // true for Int, String, etc.
	IsEnum   bool     `json:"isEnum"`
	IsModel  bool     `json:"isModel"`
}

// FieldAttribute represents an attribute on a field (e.g., @id, @unique).
type FieldAttribute struct {
	Name string                 `json:"name"` // id, unique, default, relation, map, updatedAt, etc.
	Args map[string]interface{} `json:"args,omitempty"`
}

// ModelAttribute represents a model-level attribute (e.g., @@id, @@unique, @@index, @@map).
type ModelAttribute struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// DefaultValue represents the default value of a field.
type DefaultValue struct {
	Type       DefaultValueType `json:"type"`
	Value      interface{}      `json:"value"`
	FuncName   string           `json:"funcName,omitempty"`   // autoincrement, now, uuid, cuid, etc.
	FuncArgs   []interface{}    `json:"funcArgs,omitempty"`
}

// DefaultValueType indicates the kind of default value.
type DefaultValueType int

const (
	DefaultValueLiteral  DefaultValueType = iota // Static value (1, "hello", true)
	DefaultValueFunction                          // Function call (autoincrement(), now(), uuid())
	DefaultValueEnum                              // Enum value reference
)

// Enum represents an `enum` block.
type Enum struct {
	Name   string      `json:"name"`
	Values []EnumValue `json:"values"`
	DBName string      `json:"dbName,omitempty"` // @@map("db_enum_name")
}

// EnumValue represents a single value within an enum.
type EnumValue struct {
	Name   string `json:"name"`
	DBName string `json:"dbName,omitempty"` // @map("db_value")
}

// ============================================================================
// Scalar type constants
// ============================================================================

// ScalarType defines the supported scalar types.
var ScalarTypes = map[string]bool{
	"String":   true,
	"Int":      true,
	"Float":    true,
	"Boolean":  true,
	"DateTime": true,
	"Json":     true,
	"BigInt":   true,
	"Bytes":    true,
	"Decimal":  true,
}

// IsScalarType returns true if the given type name is a scalar type.
func IsScalarType(name string) bool {
	return ScalarTypes[name]
}

// ============================================================================
// Relation helpers
// ============================================================================

// GetRelationAttribute returns the @relation attribute from a field, if present.
func (f *Field) GetRelationAttribute() *FieldAttribute {
	for i := range f.Attributes {
		if f.Attributes[i].Name == "relation" {
			return &f.Attributes[i]
		}
	}
	return nil
}

// HasAttribute returns true if the field has the named attribute.
func (f *Field) HasAttribute(name string) bool {
	for _, attr := range f.Attributes {
		if attr.Name == name {
			return true
		}
	}
	return false
}

// IsID returns true if the field has the @id attribute.
func (f *Field) IsID() bool {
	return f.HasAttribute("id")
}

// IsUnique returns true if the field has the @unique attribute.
func (f *Field) IsUnique() bool {
	return f.HasAttribute("unique")
}

// GetIDField returns the field with @id attribute, if any.
func (m *Model) GetIDField() *Field {
	for i := range m.Fields {
		if m.Fields[i].IsID() {
			return &m.Fields[i]
		}
	}
	return nil
}

// GetFieldByName returns a field by name.
func (m *Model) GetFieldByName(name string) *Field {
	for i := range m.Fields {
		if m.Fields[i].Name == name {
			return &m.Fields[i]
		}
	}
	return nil
}

// GetScalarFields returns only scalar (non-relation) fields.
func (m *Model) GetScalarFields() []Field {
	var result []Field
	for _, f := range m.Fields {
		if f.Type.IsScalar || f.Type.IsEnum {
			result = append(result, f)
		}
	}
	return result
}

// GetRelationFields returns only relation fields.
func (m *Model) GetRelationFields() []Field {
	var result []Field
	for _, f := range m.Fields {
		if f.Type.IsModel {
			result = append(result, f)
		}
	}
	return result
}
