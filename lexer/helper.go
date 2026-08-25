package lexer

import (
	"github.com/ayanozturk/go-php-parser/token"
	"unicode/utf8"
)

// isDigit returns true if the rune is an ASCII digit (PHP only uses 0-9 in numeric contexts).
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// isLetter returns true if the rune can start or be part of a PHP identifier.
// PHP's identifier grammar accepts every byte from 0x80 through 0xff, including
// invalid standalone UTF-8 bytes, so any decoded non-ASCII rune is valid here.
func isLetter(ch rune) bool {
	if ch < utf8.RuneSelf {
		return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
	}
	return true
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
