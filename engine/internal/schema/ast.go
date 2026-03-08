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

// ============================================================================
// Relation info helpers
// ============================================================================

// RelationDirection indicates how two models are related.
type RelationDirection int

const (
	// RelationBelongsTo — this model holds the FK (e.g., Post.authorId → User.id).
	RelationBelongsTo RelationDirection = iota
	// RelationHasOne — the other model holds the FK, single result.
	RelationHasOne
	// RelationHasMany — the other model holds the FK, multiple results.
	RelationHasMany
)

// RelationInfo describes a resolved relation between two models.
type RelationInfo struct {
	FieldName   string            // Field name on the source model (e.g., "posts", "author")
	TargetModel string            // Target model name (e.g., "Post", "User")
	FKFields    []string          // Foreign key field names on the FK-holding model
	RefFields   []string          // Referenced field names on the referenced model
	IsList      bool              // Whether this is a list relation
	Direction   RelationDirection // BelongsTo, HasOne, or HasMany
}

// GetRelationInfo resolves relation metadata for a relation field.
//
// Why this complexity? In Prisma-style schemas, the @relation attribute with
// fields/references only appears on ONE side of the relation (the FK-holding side).
// The other side is a "virtual" back-reference. We need to handle both directions.
func (m *Model) GetRelationInfo(fieldName string, schema *Schema) *RelationInfo {
	field := m.GetFieldByName(fieldName)
	if field == nil || !field.Type.IsModel {
		return nil
	}

	targetModel := schema.GetModelByName(field.Type.Name)
	if targetModel == nil {
		return nil
	}

	info := &RelationInfo{
		FieldName:   fieldName,
		TargetModel: field.Type.Name,
		IsList:      field.IsList,
	}

	// Case 1: This field has @relation(fields: [...], references: [...])
	// → BelongsTo direction (this model holds the FK)
	relAttr := field.GetRelationAttribute()
	if relAttr != nil {
		if fieldsArg, ok := relAttr.Args["fields"]; ok {
			if refsArg, ok := relAttr.Args["references"]; ok {
				info.FKFields = toStringSliceAST(fieldsArg)
				info.RefFields = toStringSliceAST(refsArg)
				info.Direction = RelationBelongsTo
				return info
			}
		}
	}

	// Case 2: The opposite side holds the FK — look for a field on the target
	// model whose @relation(fields: ..., references: ...) points back to us.
	for _, targetField := range targetModel.Fields {
		if !targetField.Type.IsModel || targetField.Type.Name != m.Name {
			continue
		}
		tRelAttr := targetField.GetRelationAttribute()
		if tRelAttr == nil {
			continue
		}
		fieldsArg, hasFields := tRelAttr.Args["fields"]
		refsArg, hasRefs := tRelAttr.Args["references"]
		if !hasFields || !hasRefs {
			continue
		}

		// The target model's FK fields → our referenced fields
		info.FKFields = toStringSliceAST(fieldsArg)   // FK columns on target table
		info.RefFields = toStringSliceAST(refsArg)     // PK/unique columns on source table

		if field.IsList {
			info.Direction = RelationHasMany
		} else {
			info.Direction = RelationHasOne
		}
		return info
	}

	return nil
}

// GetModelByName finds a model by name in the schema.
func (s *Schema) GetModelByName(name string) *Model {
	for i := range s.Models {
		if s.Models[i].Name == name {
			return &s.Models[i]
		}
	}
	return nil
}

// toStringSliceAST converts an interface{} to a string slice (for attribute args).
func toStringSliceAST(v interface{}) []string {
	if list, ok := v.([]interface{}); ok {
		var result []string
		for _, item := range list {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if s, ok := v.(string); ok {
		return []string{s}
	}
	return nil
}
