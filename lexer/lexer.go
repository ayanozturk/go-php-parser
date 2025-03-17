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

func (l *Lexer) readIdentifier() string {
	position := l.pos
	for isLetter(l.char) || isDigit(l.char) {
		l.readChar()
	}
	return l.input[position:l.pos]
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	pos := token.Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.pos,
	}

	switch {
	case l.char == 0:
		tok = token.Token{Type: token.T_EOF, Literal: "", Pos: pos}
	case strings.HasPrefix(l.input[l.pos:], "<?php"):
		tok = token.Token{Type: token.T_OPEN_TAG, Literal: "<?php", Pos: pos}
		l.pos += 4 // Skip the rest of <?php
		l.readChar()
	case l.char == '$':
		l.readChar()
		if isLetter(l.char) {
			ident := l.readIdentifier()
			tok = token.Token{Type: token.T_VARIABLE, Literal: "$" + ident, Pos: pos}
		}
	case l.char == '(':
		tok = token.Token{Type: token.T_LPAREN, Literal: "(", Pos: pos}
		l.readChar()
	case l.char == ')':
		tok = token.Token{Type: token.T_RPAREN, Literal: ")", Pos: pos}
		l.readChar()
	case l.char == '{':
		tok = token.Token{Type: token.T_LBRACE, Literal: "{", Pos: pos}
		l.readChar()
	case l.char == '}':
		tok = token.Token{Type: token.T_RBRACE, Literal: "}", Pos: pos}
		l.readChar()
	case l.char == ';':
		tok = token.Token{Type: token.T_SEMICOLON, Literal: ";", Pos: pos}
		l.readChar()
	case l.char == '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.T_IS_EQUAL, Literal: "==", Pos: pos}
		} else {
			tok = token.Token{Type: token.T_ASSIGN, Literal: "=", Pos: pos}
		}
		l.readChar()
	case l.char == '"':
		str := l.readString()
		tok = token.Token{Type: token.T_CONSTANT_STRING, Literal: str, Pos: pos}
		l.readChar()
	case isLetter(l.char):
		ident := l.readIdentifier()
		if ident == "function" {
			tok = token.Token{Type: token.T_FUNCTION, Literal: ident, Pos: pos}
		} else {
			tok = token.Token{Type: token.T_IDENTIFIER, Literal: ident, Pos: pos}
		}
	default:
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
