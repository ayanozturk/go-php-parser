package lexer

import (
	"go-phpcs/token"
	"strings"
	"unicode"
)

type Lexer struct {
	input   string
	pos     int
	readPos int
	char    byte
	line    int
	column  int
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 1,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.char = 0
	} else {
		l.char = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++

	if l.char == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) skipWhitespace() {
	for l.char == ' ' || l.char == '\t' || l.char == '\n' || l.char == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readString() string {
	position := l.pos + 1
	for {
		l.readChar()
		if l.char == '"' || l.char == 0 {
			break
		}
	}
	return l.input[position:l.pos]
}

func (l *Lexer) readNumber() (string, bool) {
	position := l.pos
	isFloat := false

	for isDigit(l.char) || l.char == '.' {
		if l.char == '.' {
			if isFloat { // Second decimal point
				break
			}
			isFloat = true
		}
		l.readChar()
	}

	return l.input[position:l.pos], isFloat
}

func (l *Lexer) readIdentifier() string {
	position := l.pos
	for isLetter(l.char) || isDigit(l.char) {
		l.readChar()
	}
	return l.input[position:l.pos]
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}

	l.skipWhitespace()

	switch l.char {
	case '<':
		if l.peekChar() == '?' {
			l.readChar() // consume ?
			l.readChar() // consume following char
			tok = token.Token{Type: token.T_OPEN_TAG, Literal: "<?php", Pos: pos}
		}
	case '$':
		l.readChar()
		if isLetter(l.char) {
			tok.Literal = "$" + l.readIdentifier()
			tok.Type = token.T_VARIABLE
		}
	case '(':
		tok = token.Token{Type: token.T_LPAREN, Literal: "(", Pos: pos}
		l.readChar()
	case ')':
		tok = token.Token{Type: token.T_RPAREN, Literal: ")", Pos: pos}
		l.readChar()
	case '{':
		tok = token.Token{Type: token.T_LBRACE, Literal: "{", Pos: pos}
		l.readChar()
	case '}':
		tok = token.Token{Type: token.T_RBRACE, Literal: "}", Pos: pos}
		l.readChar()
	case ';':
		tok = token.Token{Type: token.T_SEMICOLON, Literal: ";", Pos: pos}
		l.readChar()
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.T_IS_EQUAL, Literal: "==", Pos: pos}
		} else {
			tok = token.Token{Type: token.T_ASSIGN, Literal: "=", Pos: pos}
		}
		l.readChar()
	case '&':
		tok = token.Token{Type: token.T_AMPERSAND, Literal: "&", Pos: pos}
		l.readChar()
	case ',':
		tok = token.Token{Type: token.T_COMMA, Literal: ",", Pos: pos}
		l.readChar()
	case '.':
		if l.peekChar() == '.' && l.input[l.readPos+1] == '.' {
			l.readChar() // consume second .
			l.readChar() // consume third .
			tok = token.Token{Type: token.T_ELLIPSIS, Literal: "...", Pos: pos}
		}
		l.readChar()
	case '"':
		str := l.readString()
		tok = token.Token{Type: token.T_CONSTANT_STRING, Literal: str, Pos: pos}
		l.readChar()
	case 0:
		tok = token.Token{Type: token.T_EOF, Literal: "", Pos: pos}
	default:
		if isDigit(l.char) {
			num, isFloat := l.readNumber()
			if isFloat {
				tok = token.Token{Type: token.T_DNUMBER, Literal: num, Pos: pos}
			} else {
				tok = token.Token{Type: token.T_LNUMBER, Literal: num, Pos: pos}
			}
			return tok
		} else if isLetter(l.char) {
			ident := l.readIdentifier()
			switch strings.ToLower(ident) {
			case "function":
				tok = token.Token{Type: token.T_FUNCTION, Literal: ident, Pos: pos}
			case "array":
				tok = token.Token{Type: token.T_ARRAY, Literal: ident, Pos: pos}
			case "string":
				tok = token.Token{Type: token.T_STRING, Literal: ident, Pos: pos}
			case "callable":
				tok = token.Token{Type: token.T_CALLABLE, Literal: ident, Pos: pos}
			case "true":
				tok = token.Token{Type: token.T_TRUE, Literal: ident, Pos: pos}
			case "false":
				tok = token.Token{Type: token.T_FALSE, Literal: ident, Pos: pos}
			case "null":
				tok = token.Token{Type: token.T_NULL, Literal: ident, Pos: pos}
			default:
				tok = token.Token{Type: token.T_IDENTIFIER, Literal: ident, Pos: pos}
			}
			return tok
		}
		l.readChar()
	}

	return tok
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}
