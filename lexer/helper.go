package lexer

import (
	"go-phpcs/token"
	"unicode"
)

// isDigit returns true if the rune is a digit
func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}

// isLetter returns true if the rune can start or be part of a PHP identifier (supports Unicode)
func isLetter(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.In(ch, unicode.Letter, unicode.Other_ID_Start)
}

// isIdentifierStart returns true if the rune can start a PHP identifier (namespace, variable, etc.)
func isIdentifierStart(ch rune) bool {
	return isLetter(ch)
}

// PeekToken returns the next token without consuming it
func (l *Lexer) PeekToken() token.Token {
	pos := l.pos
	readPos := l.readPos
	ch := l.char
	size := l.size
	line := l.line
	column := l.column
	inString := l.inString
	heredocTokens := append([]token.Token(nil), l.heredocTokens...)
	tok := l.NextToken()
	l.pos = pos
	l.readPos = readPos
	l.char = ch
	l.size = size
	l.line = line
	l.column = column
	l.inString = inString
	l.heredocTokens = heredocTokens
	return tok
}
