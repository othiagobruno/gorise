package schema

import (
	"testing"
)

// ============================================================================
// Parser tests
// ============================================================================

const testSchema = `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "practor-client"
  output   = "./generated/client"
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  role      Role     @default(USER)
  posts     Post[]
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
  createdAt DateTime @default(now())
}

enum Role {
  USER
  ADMIN
  MODERATOR
}
`

func TestParseDatasource(t *testing.T) {
	schema, err := Parse(testSchema)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(schema.Datasources) != 1 {
		t.Fatalf("Expected 1 datasource, got %d", len(schema.Datasources))
	}

	ds := schema.Datasources[0]
	if ds.Name != "db" {
		t.Errorf("Expected datasource name 'db', got '%s'", ds.Name)
	}
	if ds.Provider != "postgresql" {
		t.Errorf("Expected provider 'postgresql', got '%s'", ds.Provider)
	}
	if !ds.IsEnvVar {
		t.Error("Expected URL to be an env var")
	}
	if ds.EnvVarName != "DATABASE_URL" {
		t.Errorf("Expected env var 'DATABASE_URL', got '%s'", ds.EnvVarName)
	}
}

func TestParseGenerator(t *testing.T) {
	schema, err := Parse(testSchema)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(schema.Generators) != 1 {
		t.Fatalf("Expected 1 generator, got %d", len(schema.Generators))
	}

	gen := schema.Generators[0]
	if gen.Name != "client" {
		t.Errorf("Expected generator name 'client', got '%s'", gen.Name)
	}
	if gen.Provider != "practor-client" {
		t.Errorf("Expected provider 'practor-client', got '%s'", gen.Provider)
	}
	if gen.Output != "./generated/client" {
		t.Errorf("Expected output './generated/client', got '%s'", gen.Output)
	}
}

func TestParseModels(t *testing.T) {
	schema, err := Parse(testSchema)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(schema.Models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(schema.Models))
	}

	// Test User model
	user := schema.Models[0]
	if user.Name != "User" {
		t.Errorf("Expected model name 'User', got '%s'", user.Name)
	}
	if len(user.Fields) != 7 {
		t.Errorf("Expected 7 fields in User, got %d", len(user.Fields))
	}

	// Test ID field
	idField := user.Fields[0]
	if idField.Name != "id" {
		t.Errorf("Expected first field 'id', got '%s'", idField.Name)
	}
	if !idField.Type.IsScalar {
		t.Error("Expected id to be scalar")
	}
	if !idField.IsID() {
		t.Error("Expected id to have @id attribute")
	}

	// Test email field
	emailField := user.Fields[1]
	if emailField.Name != "email" {
		t.Errorf("Expected second field 'email', got '%s'", emailField.Name)
	}
	if !emailField.IsUnique() {
		t.Error("Expected email to have @unique attribute")
	}

	// Test optional field
	nameField := user.Fields[2]
	if !nameField.IsOptional {
		t.Error("Expected name to be optional")
	}

	// Test Post model
	post := schema.Models[1]
	if post.Name != "Post" {
		t.Errorf("Expected model name 'Post', got '%s'", post.Name)
	}
}

func TestParseEnum(t *testing.T) {
	schema, err := Parse(testSchema)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(schema.Enums))
	}

	enum := schema.Enums[0]
	if enum.Name != "Role" {
		t.Errorf("Expected enum name 'Role', got '%s'", enum.Name)
	}
	if len(enum.Values) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(enum.Values))
	}

	expectedValues := []string{"USER", "ADMIN", "MODERATOR"}
	for i, expected := range expectedValues {
		if enum.Values[i].Name != expected {
			t.Errorf("Expected enum value '%s', got '%s'", expected, enum.Values[i].Name)
		}
	}
}

func TestParseRelations(t *testing.T) {
	schema, err := Parse(testSchema)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	ResolveFieldTypes(schema)

	// User.posts should be a list relation to Post
	user := schema.Models[0]
	postsField := user.GetFieldByName("posts")
	if postsField == nil {
		t.Fatal("Expected 'posts' field in User model")
	}
	if !postsField.IsList {
		t.Error("Expected posts to be a list")
	}
	if !postsField.Type.IsModel {
		t.Error("Expected posts type to be a model")
	}
	if postsField.Type.Name != "Post" {
		t.Errorf("Expected posts type 'Post', got '%s'", postsField.Type.Name)
	}

	// Post.author should have @relation with fields/references
	post := schema.Models[1]
	authorField := post.GetFieldByName("author")
	if authorField == nil {
		t.Fatal("Expected 'author' field in Post model")
	}
	relAttr := authorField.GetRelationAttribute()
	if relAttr == nil {
		t.Fatal("Expected @relation attribute on author field")
	}
	if _, ok := relAttr.Args["fields"]; !ok {
		t.Error("Expected 'fields' arg in @relation")
	}
	if _, ok := relAttr.Args["references"]; !ok {
		t.Error("Expected 'references' arg in @relation")
	}
}

func TestParseDefaultValues(t *testing.T) {
	schema, err := Parse(testSchema)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	user := schema.Models[0]

	// id: autoincrement()
	idField := user.GetFieldByName("id")
	if idField.DefaultValue == nil {
		t.Fatal("Expected default value for id field")
	}
	if idField.DefaultValue.Type != DefaultValueFunction {
		t.Errorf("Expected function default, got type %d", idField.DefaultValue.Type)
	}
	if idField.DefaultValue.FuncName != "autoincrement" {
		t.Errorf("Expected 'autoincrement', got '%s'", idField.DefaultValue.FuncName)
	}

	// createdAt: now()
	createdAt := user.GetFieldByName("createdAt")
	if createdAt.DefaultValue == nil {
		t.Fatal("Expected default value for createdAt")
	}
	if createdAt.DefaultValue.FuncName != "now" {
		t.Errorf("Expected 'now', got '%s'", createdAt.DefaultValue.FuncName)
	}

	// Post.published: false
	post := schema.Models[1]
	published := post.GetFieldByName("published")
	if published.DefaultValue == nil {
		t.Fatal("Expected default value for published")
	}
	if published.DefaultValue.Type != DefaultValueLiteral {
		t.Errorf("Expected literal default, got type %d", published.DefaultValue.Type)
	}
	if published.DefaultValue.Value != false {
		t.Errorf("Expected false, got %v", published.DefaultValue.Value)
	}
}

func TestValidation(t *testing.T) {
	schema, err := Parse(testSchema)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	ResolveFieldTypes(schema)
	errors := Validate(schema)

	if len(errors) != 0 {
		for _, e := range errors {
			t.Errorf("Unexpected validation error: %s", e.Error())
		}
	}
}

func TestLexerTokens(t *testing.T) {
	input := `model User { id Int @id }`
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("Lexer error: %v", err)
	}

	expected := []TokenType{TokenModel, TokenIdent, TokenLBrace, TokenIdent, TokenIdent, TokenAt, TokenIdent, TokenRBrace, TokenEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Errorf("Token %d: expected type %d, got %d ('%s')", i, exp, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestScalarTypes(t *testing.T) {
	scalars := []string{"String", "Int", "Float", "Boolean", "DateTime", "Json", "BigInt", "Bytes", "Decimal"}
	for _, s := range scalars {
		if !IsScalarType(s) {
			t.Errorf("Expected '%s' to be a scalar type", s)
		}
	}
	if IsScalarType("User") {
		t.Error("'User' should not be a scalar type")
	}
}
