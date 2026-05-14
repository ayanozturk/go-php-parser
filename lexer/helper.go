package lexer

import (
	"go-phpcs/token"
	"unicode"
)

// isDigit returns true if the rune is an ASCII digit (PHP only uses 0-9 in numeric contexts).
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// isLetter returns true if the rune can start or be part of a PHP identifier (supports Unicode).
// ASCII fast path avoids unicode table lookups for the common case.
func isLetter(ch rune) bool {
	if ch < 128 {
		return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
	}
	return unicode.IsLetter(ch) || unicode.In(ch, unicode.Other_ID_Start)
}

// isIdentifierStart returns true if the rune can start a PHP identifier (namespace, variable, etc.)
func isIdentifierStart(ch rune) bool {
	return isLetter(ch)
}

// PeekToken returns the next token without consuming it.
// Uses a single-token lookahead cache — no state save/restore or slice copy.
func (l *Lexer) PeekToken() token.Token {
	if !l.hasPeeked {
		l.peekedToken = l.scanToken()
		l.hasPeeked = true
	}
	return l.peekedToken
}
