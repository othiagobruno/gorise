package schema

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ============================================================================
// Token types
// ============================================================================

// TokenType represents the type of a lexer token.
type TokenType int

const (
	// Special
	TokenEOF TokenType = iota
	TokenError

	// Literals
	TokenIdent        // identifier
	TokenString       // "quoted string"
	TokenNumber       // 123, 4.56
	TokenBool         // true, false

	// Delimiters
	TokenLBrace       // {
	TokenRBrace       // }
	TokenLParen       // (
	TokenRParen       // )
	TokenLBracket     // [
	TokenRBracket     // ]
	TokenComma        // ,
	TokenDot          // .
	TokenColon        // :
	TokenQuestion     // ?
	TokenAt           // @
	TokenDoubleAt     // @@
	TokenEquals       // =
	TokenNewline      // \n

	// Keywords
	TokenDatasource   // datasource
	TokenGenerator    // generator
	TokenModel        // model
	TokenEnum         // enum
	TokenType_        // type (for composite types)

	// Comment
	TokenComment      // // comment or /// doc comment
)

// Token represents a single lexer token.
type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Column  int
}

// String returns a human-readable representation of the token.
func (t Token) String() string {
	switch t.Type {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return fmt.Sprintf("ERROR(%s)", t.Value)
	default:
		return fmt.Sprintf("%d(%s)", t.Type, t.Value)
	}
}

// ============================================================================
// Lexer
// ============================================================================

// Lexer tokenizes a Practor schema string.
type Lexer struct {
	input   string
	pos     int
	line    int
	col     int
	tokens  []Token
}

// NewLexer creates a new Lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
		line:  1,
		col:   1,
	}
}

// Tokenize processes the entire input and returns all tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return l.tokens, nil
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r := l.peek()
		if r == ' ' || r == '\t' || r == '\r' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) nextToken() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Line: l.line, Column: l.col}, nil
	}

	line := l.line
	col := l.col
	r := l.peek()

	// Newlines
	if r == '\n' {
		l.advance()
		return Token{Type: TokenNewline, Value: "\n", Line: line, Column: col}, nil
	}

	// Comments
	if r == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '/' {
		return l.lexComment(line, col), nil
	}

	// Strings
	if r == '"' {
		return l.lexString(line, col)
	}

	// Numbers
	if unicode.IsDigit(r) || (r == '-' && l.pos+1 < len(l.input) && unicode.IsDigit(rune(l.input[l.pos+1]))) {
		return l.lexNumber(line, col), nil
	}

	// Identifiers and keywords
	if unicode.IsLetter(r) || r == '_' {
		return l.lexIdentifier(line, col), nil
	}

	// Single/double character tokens
	l.advance()

	switch r {
	case '{':
		return Token{Type: TokenLBrace, Value: "{", Line: line, Column: col}, nil
	case '}':
		return Token{Type: TokenRBrace, Value: "}", Line: line, Column: col}, nil
	case '(':
		return Token{Type: TokenLParen, Value: "(", Line: line, Column: col}, nil
	case ')':
		return Token{Type: TokenRParen, Value: ")", Line: line, Column: col}, nil
	case '[':
		return Token{Type: TokenLBracket, Value: "[", Line: line, Column: col}, nil
	case ']':
		return Token{Type: TokenRBracket, Value: "]", Line: line, Column: col}, nil
	case ',':
		return Token{Type: TokenComma, Value: ",", Line: line, Column: col}, nil
	case '.':
		return Token{Type: TokenDot, Value: ".", Line: line, Column: col}, nil
	case ':':
		return Token{Type: TokenColon, Value: ":", Line: line, Column: col}, nil
	case '?':
		return Token{Type: TokenQuestion, Value: "?", Line: line, Column: col}, nil
	case '=':
		return Token{Type: TokenEquals, Value: "=", Line: line, Column: col}, nil
	case '@':
		if l.peek() == '@' {
			l.advance()
			return Token{Type: TokenDoubleAt, Value: "@@", Line: line, Column: col}, nil
		}
		return Token{Type: TokenAt, Value: "@", Line: line, Column: col}, nil
	}

	return Token{Type: TokenError, Value: string(r), Line: line, Column: col},
		fmt.Errorf("unexpected character '%c' at line %d, column %d", r, line, col)
}

func (l *Lexer) lexComment(line, col int) Token {
	start := l.pos
	// Skip the //
	l.advance()
	l.advance()

	// Check for doc comment (///)
	isDoc := false
	if l.peek() == '/' {
		l.advance()
		isDoc = true
	}
	_ = isDoc

	// Read until end of line
	for l.pos < len(l.input) && l.peek() != '\n' {
		l.advance()
	}

	return Token{
		Type:  TokenComment,
		Value: strings.TrimSpace(l.input[start:l.pos]),
		Line:  line,
		Column: col,
	}
}

func (l *Lexer) lexString(line, col int) (Token, error) {
	l.advance() // skip opening "
	var sb strings.Builder

	for {
		if l.pos >= len(l.input) {
			return Token{Type: TokenError, Line: line, Column: col},
				fmt.Errorf("unterminated string at line %d, column %d", line, col)
		}

		r := l.advance()

		if r == '"' {
			break
		}

		if r == '\\' {
			next := l.advance()
			switch next {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(next)
			}
			continue
		}

		sb.WriteRune(r)
	}

	return Token{Type: TokenString, Value: sb.String(), Line: line, Column: col}, nil
}

func (l *Lexer) lexNumber(line, col int) Token {
	start := l.pos
	if l.peek() == '-' {
		l.advance()
	}

	for l.pos < len(l.input) && unicode.IsDigit(l.peek()) {
		l.advance()
	}

	// Float
	if l.pos < len(l.input) && l.peek() == '.' {
		l.advance()
		for l.pos < len(l.input) && unicode.IsDigit(l.peek()) {
			l.advance()
		}
	}

	return Token{Type: TokenNumber, Value: l.input[start:l.pos], Line: line, Column: col}
}

func (l *Lexer) lexIdentifier(line, col int) Token {
	start := l.pos

	for l.pos < len(l.input) {
		r := l.peek()
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			l.advance()
		} else {
			break
		}
	}

	value := l.input[start:l.pos]

	// Check for keywords
	switch value {
	case "datasource":
		return Token{Type: TokenDatasource, Value: value, Line: line, Column: col}
	case "generator":
		return Token{Type: TokenGenerator, Value: value, Line: line, Column: col}
	case "model":
		return Token{Type: TokenModel, Value: value, Line: line, Column: col}
	case "enum":
		return Token{Type: TokenEnum, Value: value, Line: line, Column: col}
	case "type":
		return Token{Type: TokenType_, Value: value, Line: line, Column: col}
	case "true", "false":
		return Token{Type: TokenBool, Value: value, Line: line, Column: col}
	}

	return Token{Type: TokenIdent, Value: value, Line: line, Column: col}
}
