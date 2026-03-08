package schema

import (
	"fmt"
	"strconv"
	"strings"
)

// ============================================================================
// Parser — Recursive descent parser for Practor Schema Language
// ============================================================================

// Parser parses a list of tokens into a Schema AST.
type Parser struct {
	tokens  []Token
	pos     int
	schema  *Schema
	errors  []string
}

// NewParser creates a new Parser from a list of tokens.
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
		schema: &Schema{},
	}
}

// Parse parses the token stream and returns a Schema AST.
func Parse(input string) (*Schema, error) {
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexer error: %w", err)
	}

	parser := NewParser(tokens)
	return parser.Parse()
}

// Parse processes all tokens and produces the Schema.
func (p *Parser) Parse() (*Schema, error) {
	p.skipNewlines()

	for !p.isAtEnd() {
		p.skipNewlines()
		if p.isAtEnd() {
			break
		}

		// Skip comments at top level
		if p.check(TokenComment) {
			p.advance()
			continue
		}

		switch {
		case p.check(TokenDatasource):
			ds, err := p.parseDatasource()
			if err != nil {
				return nil, err
			}
			p.schema.Datasources = append(p.schema.Datasources, *ds)

		case p.check(TokenGenerator):
			gen, err := p.parseGenerator()
			if err != nil {
				return nil, err
			}
			p.schema.Generators = append(p.schema.Generators, *gen)

		case p.check(TokenModel):
			model, err := p.parseModel()
			if err != nil {
				return nil, err
			}
			p.schema.Models = append(p.schema.Models, *model)

		case p.check(TokenEnum):
			enum, err := p.parseEnum()
			if err != nil {
				return nil, err
			}
			p.schema.Enums = append(p.schema.Enums, *enum)

		default:
			tok := p.current()
			return nil, fmt.Errorf("unexpected token '%s' at line %d, column %d", tok.Value, tok.Line, tok.Column)
		}
	}

	return p.schema, nil
}

// ============================================================================
// Block parsers
// ============================================================================

func (p *Parser) parseDatasource() (*Datasource, error) {
	p.expect(TokenDatasource) // consume 'datasource'
	p.skipNewlines()

	name, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected datasource name: %w", err)
	}

	p.skipNewlines()
	if _, err := p.expectAndReturn(TokenLBrace); err != nil {
		return nil, fmt.Errorf("expected '{' after datasource name: %w", err)
	}

	ds := &Datasource{
		Name:       name.Value,
		Properties: make(map[string]string),
	}

	p.skipNewlines()
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipComments()
		if p.check(TokenRBrace) {
			break
		}

		key, err := p.expectAndReturn(TokenIdent)
		if err != nil {
			return nil, fmt.Errorf("expected property key in datasource: %w", err)
		}

		if _, err := p.expectAndReturn(TokenEquals); err != nil {
			return nil, fmt.Errorf("expected '=' after property key '%s': %w", key.Value, err)
		}

		// Parse the value — can be a string or env("...") function call
		value, isEnv, envVar, err := p.parsePropertyValue()
		if err != nil {
			return nil, err
		}

		switch key.Value {
		case "provider":
			ds.Provider = value
		case "url":
			ds.URL = value
			ds.IsEnvVar = isEnv
			ds.EnvVarName = envVar
		default:
			ds.Properties[key.Value] = value
		}

		p.skipNewlines()
	}

	if _, err := p.expectAndReturn(TokenRBrace); err != nil {
		return nil, fmt.Errorf("expected '}' to close datasource block: %w", err)
	}

	return ds, nil
}

func (p *Parser) parseGenerator() (*Generator, error) {
	p.expect(TokenGenerator)
	p.skipNewlines()

	name, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected generator name: %w", err)
	}

	p.skipNewlines()
	if _, err := p.expectAndReturn(TokenLBrace); err != nil {
		return nil, fmt.Errorf("expected '{': %w", err)
	}

	gen := &Generator{
		Name:       name.Value,
		Properties: make(map[string]string),
	}

	p.skipNewlines()
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipComments()
		if p.check(TokenRBrace) {
			break
		}

		key, err := p.expectAndReturn(TokenIdent)
		if err != nil {
			return nil, fmt.Errorf("expected property key in generator: %w", err)
		}

		if _, err := p.expectAndReturn(TokenEquals); err != nil {
			return nil, fmt.Errorf("expected '=': %w", err)
		}

		value, _, _, err := p.parsePropertyValue()
		if err != nil {
			return nil, err
		}

		switch key.Value {
		case "provider":
			gen.Provider = value
		case "output":
			gen.Output = value
		default:
			gen.Properties[key.Value] = value
		}

		p.skipNewlines()
	}

	if _, err := p.expectAndReturn(TokenRBrace); err != nil {
		return nil, fmt.Errorf("expected '}': %w", err)
	}

	return gen, nil
}

func (p *Parser) parseModel() (*Model, error) {
	p.expect(TokenModel)
	p.skipNewlines()

	name, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected model name: %w", err)
	}

	p.skipNewlines()
	if _, err := p.expectAndReturn(TokenLBrace); err != nil {
		return nil, fmt.Errorf("expected '{': %w", err)
	}

	model := &Model{
		Name: name.Value,
	}

	p.skipNewlines()
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipComments()
		if p.check(TokenRBrace) {
			break
		}

		// Model-level attributes (@@)
		if p.check(TokenDoubleAt) {
			attr, err := p.parseModelAttribute()
			if err != nil {
				return nil, err
			}
			model.Attributes = append(model.Attributes, *attr)
			p.skipNewlines()
			continue
		}

		field, err := p.parseField()
		if err != nil {
			return nil, fmt.Errorf("error parsing field in model '%s': %w", model.Name, err)
		}
		model.Fields = append(model.Fields, *field)
		p.skipNewlines()
	}

	if _, err := p.expectAndReturn(TokenRBrace); err != nil {
		return nil, fmt.Errorf("expected '}': %w", err)
	}

	return model, nil
}

func (p *Parser) parseField() (*Field, error) {
	nameTok, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected field name: %w", err)
	}

	// Parse the field type
	typeTok, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected field type for '%s': %w", nameTok.Value, err)
	}

	field := &Field{
		Name: nameTok.Value,
		Type: FieldType{
			Name:     typeTok.Value,
			IsScalar: IsScalarType(typeTok.Value),
		},
	}

	// Check for list modifier []
	if p.check(TokenLBracket) {
		p.advance() // [
		if _, err := p.expectAndReturn(TokenRBracket); err != nil {
			return nil, fmt.Errorf("expected ']' for list type: %w", err)
		}
		field.IsList = true
	}

	// Check for optional modifier ?
	if p.check(TokenQuestion) {
		p.advance()
		field.IsOptional = true
	}

	// Parse field attributes (@id, @unique, @default, @relation, @map, @updatedAt)
	for p.check(TokenAt) {
		attr, err := p.parseFieldAttribute()
		if err != nil {
			return nil, fmt.Errorf("error parsing attribute for field '%s': %w", field.Name, err)
		}
		field.Attributes = append(field.Attributes, *attr)

		// Handle @default — extract the default value for convenience
		if attr.Name == "default" {
			field.DefaultValue = p.extractDefaultValue(attr)
		}
	}

	return field, nil
}

func (p *Parser) parseFieldAttribute() (*FieldAttribute, error) {
	p.expect(TokenAt) // consume @

	nameTok, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected attribute name: %w", err)
	}

	attr := &FieldAttribute{
		Name: nameTok.Value,
		Args: make(map[string]interface{}),
	}

	// Parse arguments if present
	if p.check(TokenLParen) {
		p.advance() // (
		args, err := p.parseAttributeArgs()
		if err != nil {
			return nil, err
		}
		attr.Args = args
	}

	return attr, nil
}

func (p *Parser) parseModelAttribute() (*ModelAttribute, error) {
	p.expect(TokenDoubleAt) // consume @@

	nameTok, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected model attribute name: %w", err)
	}

	attr := &ModelAttribute{
		Name: nameTok.Value,
		Args: make(map[string]interface{}),
	}

	if p.check(TokenLParen) {
		p.advance()
		args, err := p.parseAttributeArgs()
		if err != nil {
			return nil, err
		}
		attr.Args = args
	}

	return attr, nil
}

func (p *Parser) parseAttributeArgs() (map[string]interface{}, error) {
	args := make(map[string]interface{})
	argIndex := 0

	for !p.check(TokenRParen) && !p.isAtEnd() {
		if argIndex > 0 {
			if p.check(TokenComma) {
				p.advance()
			}
		}

		// Check for named argument: name: value
		if p.check(TokenIdent) && p.peekNext().Type == TokenColon {
			name := p.advance()
			p.advance() // skip :
			value, err := p.parseArgValue()
			if err != nil {
				return nil, err
			}
			args[name.Value] = value
		} else {
			// Positional argument
			value, err := p.parseArgValue()
			if err != nil {
				return nil, err
			}
			args[fmt.Sprintf("_%d", argIndex)] = value
		}

		argIndex++
	}

	if _, err := p.expectAndReturn(TokenRParen); err != nil {
		return nil, fmt.Errorf("expected ')': %w", err)
	}

	return args, nil
}

func (p *Parser) parseArgValue() (interface{}, error) {
	tok := p.current()

	switch tok.Type {
	case TokenString:
		p.advance()
		return tok.Value, nil

	case TokenNumber:
		p.advance()
		if strings.Contains(tok.Value, ".") {
			f, _ := strconv.ParseFloat(tok.Value, 64)
			return f, nil
		}
		i, _ := strconv.ParseInt(tok.Value, 10, 64)
		return i, nil

	case TokenBool:
		p.advance()
		return tok.Value == "true", nil

	case TokenIdent:
		// Could be: function call like autoincrement(), now(), uuid()
		// or enum reference, or field reference
		name := tok.Value
		p.advance()

		if p.check(TokenLParen) {
			p.advance() // (
			// Parse function args
			var funcArgs []interface{}
			for !p.check(TokenRParen) && !p.isAtEnd() {
				if len(funcArgs) > 0 && p.check(TokenComma) {
					p.advance()
				}
				arg, err := p.parseArgValue()
				if err != nil {
					return nil, err
				}
				funcArgs = append(funcArgs, arg)
			}
			p.advance() // )
			return map[string]interface{}{
				"_func": name,
				"_args": funcArgs,
			}, nil
		}

		return name, nil

	case TokenLBracket:
		// Array value [field1, field2]
		p.advance() // [
		var items []interface{}
		for !p.check(TokenRBracket) && !p.isAtEnd() {
			if len(items) > 0 && p.check(TokenComma) {
				p.advance()
			}
			item, err := p.parseArgValue()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if _, err := p.expectAndReturn(TokenRBracket); err != nil {
			return nil, fmt.Errorf("expected ']': %w", err)
		}
		return items, nil

	default:
		return nil, fmt.Errorf("unexpected token '%s' at line %d, column %d", tok.Value, tok.Line, tok.Column)
	}
}

func (p *Parser) parseEnum() (*Enum, error) {
	p.expect(TokenEnum)
	p.skipNewlines()

	name, err := p.expectAndReturn(TokenIdent)
	if err != nil {
		return nil, fmt.Errorf("expected enum name: %w", err)
	}

	p.skipNewlines()
	if _, err := p.expectAndReturn(TokenLBrace); err != nil {
		return nil, fmt.Errorf("expected '{': %w", err)
	}

	enum := &Enum{Name: name.Value}

	p.skipNewlines()
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipComments()
		if p.check(TokenRBrace) {
			break
		}

		valueTok, err := p.expectAndReturn(TokenIdent)
		if err != nil {
			return nil, fmt.Errorf("expected enum value: %w", err)
		}

		ev := EnumValue{Name: valueTok.Value}

		// Check for @map attribute
		if p.check(TokenAt) {
			p.advance()
			mapTok, err := p.expectAndReturn(TokenIdent)
			if err == nil && mapTok.Value == "map" {
				if p.check(TokenLParen) {
					p.advance()
					valTok, err := p.expectAndReturn(TokenString)
					if err == nil {
						ev.DBName = valTok.Value
					}
					p.expectAndReturn(TokenRParen)
				}
			}
		}

		enum.Values = append(enum.Values, ev)
		p.skipNewlines()
	}

	if _, err := p.expectAndReturn(TokenRBrace); err != nil {
		return nil, fmt.Errorf("expected '}': %w", err)
	}

	return enum, nil
}

// ============================================================================
// Property value parsing (for datasource/generator blocks)
// ============================================================================

func (p *Parser) parsePropertyValue() (value string, isEnv bool, envVar string, err error) {
	// env("VAR_NAME")
	if p.check(TokenIdent) && p.current().Value == "env" {
		p.advance() // env
		if _, err := p.expectAndReturn(TokenLParen); err != nil {
			return "", false, "", err
		}
		strTok, err := p.expectAndReturn(TokenString)
		if err != nil {
			return "", false, "", fmt.Errorf("expected string in env(): %w", err)
		}
		if _, err := p.expectAndReturn(TokenRParen); err != nil {
			return "", false, "", err
		}
		return fmt.Sprintf("env(\"%s\")", strTok.Value), true, strTok.Value, nil
	}

	// Regular string
	if p.check(TokenString) {
		tok := p.advance()
		return tok.Value, false, "", nil
	}

	// Boolean
	if p.check(TokenBool) {
		tok := p.advance()
		return tok.Value, false, "", nil
	}

	// Identifier
	if p.check(TokenIdent) {
		tok := p.advance()
		return tok.Value, false, "", nil
	}

	return "", false, "", fmt.Errorf("expected property value at line %d", p.current().Line)
}

// ============================================================================
// Default value extraction
// ============================================================================

func (p *Parser) extractDefaultValue(attr *FieldAttribute) *DefaultValue {
	// The first positional arg (_0) holds the default value
	val, ok := attr.Args["_0"]
	if !ok {
		return nil
	}

	// Check if it's a function call
	if m, ok := val.(map[string]interface{}); ok {
		if funcName, ok := m["_func"].(string); ok {
			dv := &DefaultValue{
				Type:     DefaultValueFunction,
				FuncName: funcName,
			}
			if args, ok := m["_args"].([]interface{}); ok {
				dv.FuncArgs = args
			}
			return dv
		}
	}

	// Literal value
	return &DefaultValue{
		Type:  DefaultValueLiteral,
		Value: val,
	}
}

// ============================================================================
// Token navigation helpers
// ============================================================================

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekNext() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) check(t TokenType) bool {
	return p.current().Type == t
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) expect(_ TokenType) {
	p.advance()
}

func (p *Parser) expectAndReturn(t TokenType) (Token, error) {
	tok := p.current()
	if tok.Type != t {
		return tok, fmt.Errorf("expected token type %d, got '%s' (%d) at line %d, column %d",
			t, tok.Value, tok.Type, tok.Line, tok.Column)
	}
	p.advance()
	return tok, nil
}

func (p *Parser) isAtEnd() bool {
	return p.current().Type == TokenEOF
}

func (p *Parser) skipNewlines() {
	for p.check(TokenNewline) || p.check(TokenComment) {
		p.advance()
	}
}

func (p *Parser) skipComments() {
	for p.check(TokenComment) || p.check(TokenNewline) {
		p.advance()
	}
}
