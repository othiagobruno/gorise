package schema

import (
	"fmt"
	"strings"
)

// ============================================================================
// Validator — Validates a parsed Schema AST for semantic correctness
// ============================================================================

// ValidationError represents a schema validation error.
type ValidationError struct {
	Message string
	Model   string
	Field   string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("Validation error in model '%s', field '%s': %s", e.Model, e.Field, e.Message)
	}
	if e.Model != "" {
		return fmt.Sprintf("Validation error in model '%s': %s", e.Model, e.Message)
	}
	return fmt.Sprintf("Validation error: %s", e.Message)
}

// Validate checks the schema for semantic correctness.
func Validate(schema *Schema) []ValidationError {
	v := &validator{
		schema:    schema,
		modelMap:  make(map[string]*Model),
		enumMap:   make(map[string]*Enum),
	}

	// Build lookup maps
	for i := range schema.Models {
		v.modelMap[schema.Models[i].Name] = &schema.Models[i]
	}
	for i := range schema.Enums {
		v.enumMap[schema.Enums[i].Name] = &schema.Enums[i]
	}

	v.validateDatasources()
	v.validateModels()
	v.validateRelations()

	return v.errors
}

type validator struct {
	schema   *Schema
	modelMap map[string]*Model
	enumMap  map[string]*Enum
	errors   []ValidationError
}

func (v *validator) addError(model, field, message string) {
	v.errors = append(v.errors, ValidationError{
		Message: message,
		Model:   model,
		Field:   field,
	})
}

func (v *validator) validateDatasources() {
	if len(v.schema.Datasources) == 0 {
		v.addError("", "", "schema must have at least one datasource")
		return
	}

	for _, ds := range v.schema.Datasources {
		if ds.Provider == "" {
			v.addError("", "", fmt.Sprintf("datasource '%s' must have a provider", ds.Name))
		}
		validProviders := map[string]bool{
			"postgresql": true, "postgres": true,
			"mysql": true, "sqlite": true,
			"sqlserver": true, "mongodb": true,
			"cockroachdb": true,
		}
		if !validProviders[strings.ToLower(ds.Provider)] {
			v.addError("", "", fmt.Sprintf("invalid provider '%s' in datasource '%s'", ds.Provider, ds.Name))
		}
		if ds.URL == "" {
			v.addError("", "", fmt.Sprintf("datasource '%s' must have a url", ds.Name))
		}
	}
}

func (v *validator) validateModels() {
	for _, model := range v.schema.Models {
		// Every model must have at least one @id field or @@id compound key
		hasID := false
		for _, field := range model.Fields {
			if field.IsID() {
				hasID = true
				break
			}
		}

		// Check for @@id compound key
		if !hasID {
			for _, attr := range model.Attributes {
				if attr.Name == "id" {
					hasID = true
					break
				}
			}
		}

		if !hasID {
			v.addError(model.Name, "", "model must have at least one field with @id or a @@id compound key")
		}

		// Validate each field
		for _, field := range model.Fields {
			v.validateField(&model, &field)
		}
	}
}

func (v *validator) validateField(model *Model, field *Field) {
	// Resolve field type — must be a scalar, enum, or another model
	if !field.Type.IsScalar {
		if _, isEnum := v.enumMap[field.Type.Name]; isEnum {
			field.Type.IsEnum = true
		} else if _, isModel := v.modelMap[field.Type.Name]; isModel {
			field.Type.IsModel = true
		} else {
			v.addError(model.Name, field.Name,
				fmt.Sprintf("unknown type '%s' — not a scalar, enum, or model", field.Type.Name))
		}
	}

	// List fields cannot be optional
	if field.IsList && field.IsOptional {
		v.addError(model.Name, field.Name, "list fields cannot be optional (use empty list instead)")
	}

	// Validate @default value type matches field type
	if field.DefaultValue != nil {
		v.validateDefaultValue(model.Name, field)
	}
}

func (v *validator) validateDefaultValue(modelName string, field *Field) {
	dv := field.DefaultValue

	if dv.Type == DefaultValueFunction {
		validFuncs := map[string]bool{
			"autoincrement": true, "now": true, "uuid": true,
			"cuid": true, "dbgenerated": true, "auto": true,
		}
		if !validFuncs[dv.FuncName] {
			v.addError(modelName, field.Name,
				fmt.Sprintf("unknown default function '%s()'", dv.FuncName))
		}

		// autoincrement() only for Int/BigInt
		if dv.FuncName == "autoincrement" && field.Type.Name != "Int" && field.Type.Name != "BigInt" {
			v.addError(modelName, field.Name,
				"autoincrement() can only be used with Int or BigInt fields")
		}

		// now() only for DateTime
		if dv.FuncName == "now" && field.Type.Name != "DateTime" {
			v.addError(modelName, field.Name,
				"now() can only be used with DateTime fields")
		}

		// uuid()/cuid() only for String
		if (dv.FuncName == "uuid" || dv.FuncName == "cuid") && field.Type.Name != "String" {
			v.addError(modelName, field.Name,
				fmt.Sprintf("%s() can only be used with String fields", dv.FuncName))
		}
	}
}

func (v *validator) validateRelations() {
	for _, model := range v.schema.Models {
		for _, field := range model.Fields {
			if !field.Type.IsModel {
				continue
			}

			relAttr := field.GetRelationAttribute()

			// Non-list relation fields must have a @relation attribute with fields/references
			if !field.IsList && relAttr != nil {
				if _, ok := relAttr.Args["fields"]; !ok {
					v.addError(model.Name, field.Name,
						"@relation must include 'fields' argument for non-list relation fields")
				}
				if _, ok := relAttr.Args["references"]; !ok {
					v.addError(model.Name, field.Name,
						"@relation must include 'references' argument for non-list relation fields")
				}
			}

			// Check that the related model exists
			relatedModel, exists := v.modelMap[field.Type.Name]
			if !exists {
				v.addError(model.Name, field.Name,
					fmt.Sprintf("related model '%s' not found", field.Type.Name))
				continue
			}

			// Validate back-relation exists
			hasBackRelation := false
			for _, rf := range relatedModel.Fields {
				if rf.Type.Name == model.Name {
					hasBackRelation = true
					break
				}
			}
			if !hasBackRelation {
				v.addError(model.Name, field.Name,
					fmt.Sprintf("missing back-relation in model '%s' for relation to '%s'",
						relatedModel.Name, model.Name))
			}
		}
	}
}

// ResolveFieldTypes updates field types with enum/model flags based on schema context.
// Should be called after parsing and before validation for accurate type resolution.
func ResolveFieldTypes(schema *Schema) {
	modelNames := make(map[string]bool)
	enumNames := make(map[string]bool)

	for _, m := range schema.Models {
		modelNames[m.Name] = true
	}
	for _, e := range schema.Enums {
		enumNames[e.Name] = true
	}

	for i := range schema.Models {
		for j := range schema.Models[i].Fields {
			field := &schema.Models[i].Fields[j]
			if !field.Type.IsScalar {
				if enumNames[field.Type.Name] {
					field.Type.IsEnum = true
				} else if modelNames[field.Type.Name] {
					field.Type.IsModel = true
				}
			}
		}
	}
}
