package lexer

import (
	"unicode"
	"unicode/utf8"
)

// Token represents a single token from the PHP code
type Token struct {
	Type    TokenType
	Content string
	Line    int
	Pos     int
}

// TokenType defines the type of a token
type TokenType int

// Token type constants
const (
	T_INVALID TokenType = iota
	T_EOF
	T_WHITESPACE
	T_OPEN_TAG                 // <?php
	T_CLOSE_TAG                // ?>
	T_STRING                   // identifiers
	T_VARIABLE                 // $var
	T_LNUMBER                  // 123
	T_DNUMBER                  // 123.45
	T_CONSTANT_ENCAPSED_STRING // 'string' or "string"
	// ... more token types would be added here
)

// Lexer converts PHP source code into tokens
type Lexer struct {
	source  string
	pos     int
	line    int
	current rune
}

// New creates a new lexer instance
func New(source string) *Lexer {
	l := &Lexer{
		source: source,
		line:   1,
	}
	l.advance() // Initialize with first character
	return l
}

// advance moves to the next character in the input
func (l *Lexer) advance() {
	if l.pos >= len(l.source) {
		l.current = 0 // EOF
		return
	}

	r, width := utf8.DecodeRuneInString(l.source[l.pos:])
	l.current = r
	l.pos += width

	if r == '\n' {
		l.line++
	}
}

// GetNextToken returns the next token from the input
func (l *Lexer) GetNextToken() Token {
	// Skip whitespace
	if unicode.IsSpace(l.current) {
		return l.lexWhitespace()
	}

	// Check for PHP open tag
	if l.current == '<' && l.peek() == '?' {
		return l.lexOpenTag()
	}

	// Handle variables
	if l.current == '$' {
		return l.lexVariable()
	}

	// Handle numbers
	if unicode.IsDigit(l.current) {
		return l.lexNumber()
	}

	// Handle identifiers
	if isIdentifierStart(l.current) {
		return l.lexIdentifier()
	}

	// Handle strings
	if l.current == '\'' || l.current == '"' {
		return l.lexString()
	}

	// EOF
	if l.current == 0 {
		return Token{Type: T_EOF, Line: l.line, Pos: l.pos}
	}

	// Other characters
	tok := Token{Type: T_INVALID, Content: string(l.current), Line: l.line, Pos: l.pos}
	l.advance()
	return tok
}

// peek returns the next character without advancing
func (l *Lexer) peek() rune {
	if l.pos >= len(l.source) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[l.pos:])
	return r
}

// Helper lexing functions would be implemented below
func (l *Lexer) lexWhitespace() Token {
	// Implementation for lexing whitespace
	// ...existing code...
	return Token{}
}

func (l *Lexer) lexOpenTag() Token {
	// Implementation for lexing PHP open tag
	// ...existing code...
	return Token{}
}

func (l *Lexer) lexVariable() Token {
	// Implementation for lexing variables
	// ...existing code...
	return Token{}
}

func (l *Lexer) lexNumber() Token {
	// Implementation for lexing numbers
	// ...existing code...
	return Token{}
}

func (l *Lexer) lexIdentifier() Token {
	// Implementation for lexing identifiers
	// ...existing code...
	return Token{}
}

func (l *Lexer) lexString() Token {
	// Implementation for lexing strings
	// ...existing code...
	return Token{}
}

// Helper function to check if a character can start an identifier
func isIdentifierStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}
