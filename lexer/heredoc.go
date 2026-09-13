package lexer

import (
	"bytes"
	"unicode/utf8"

	"github.com/ayanozturk/go-php-parser/token"
)

// queueHeredocTokens starts a heredoc/nowdoc. T_START_* keeps the exact opener
// text including the trailing newline (Zend). Body bytes are preserved; indent
// stripping is a syntax/semantic property, not done in the lexer.
func (l *Lexer) queueHeredocTokens(pos token.Position) {
	startOff := l.pos
	l.readChar() // <
	l.readChar() // <
	l.readChar() // <

	// Optional whitespace between <<< and the label (not for indented form's body).
	for l.char == ' ' || l.char == '\t' {
		l.readChar()
	}

	label, isNowdoc := l.readHeredocIdentifier()
	if label == "" {
		l.queueToken(token.Token{
			Type:    token.T_ILLEGAL,
			Literal: "Missing heredoc/nowdoc identifier",
			Pos:     pos,
			End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
		})
		return
	}

	// Consume rest of line including newline into the start token (Zend includes it).
	for !l.atEOF() && l.char != '\n' {
		if l.char == '\r' && l.peekChar() == '\n' {
			l.readChar()
			break
		}
		if l.char == '\r' {
			break
		}
		l.readChar()
	}
	if l.char == '\r' {
		l.readChar()
		if l.char == '\n' {
			l.readChar()
		}
	} else if l.char == '\n' {
		l.readChar()
	}

	startType := token.T_START_HEREDOC
	if isNowdoc {
		startType = token.T_START_NOWDOC
	}
	startTok := token.Token{
		Type:    startType,
		Literal: l.text(startOff, l.pos),
		Pos:     pos,
		End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
	}

	l.heredocLabel = label
	l.heredocNowdoc = isNowdoc
	l.encapsed = encapsedHeredoc
	// Emit start first via return path: caller does nextHeredocToken after queue.
	// We return start via queue then body.
	l.heredocTokens = []token.Token{startTok}
	l.queueEncapsedBody(isNowdoc)
}

func (l *Lexer) readHeredocIdentifier() (string, bool) {
	if l.char == '\'' || l.char == '"' {
		quote := l.char
		l.readChar()
		start := l.pos
		for l.char != quote && !l.atEOF() {
			l.readChar()
		}
		identifier := l.text(start, l.pos)
		isNowdoc := quote == '\''
		if l.char == quote {
			l.readChar()
		}
		return identifier, isNowdoc
	}
	start := l.pos
	for isLetter(l.char) || isDigit(l.char) {
		l.readChar()
	}
	return l.text(start, l.pos), false
}

// matchHeredocTerminator reports whether the current position is a valid
// terminator line: optional indent + exact label + non-identifier char.
// Returns indent text, label, and ok. Does not consume input.
func (l *Lexer) matchHeredocTerminator() (indent, label string, ok bool) {
	if l.heredocLabel == "" {
		return "", "", false
	}
	// Terminators are only recognized at the beginning of a line.
	if l.pos > 0 && l.input[l.pos-1] != '\n' {
		return "", "", false
	}
	identifierPos := l.pos
	for identifierPos < len(l.input) && (l.input[identifierPos] == ' ' || l.input[identifierPos] == '\t') {
		identifierPos++
	}
	label = l.heredocLabel
	if identifierPos+len(label) > len(l.input) || !bytes.Equal(l.input[identifierPos:identifierPos+len(label)], []byte(label)) {
		return "", "", false
	}
	var nextChar rune
	nextPos := identifierPos + len(label)
	if nextPos < len(l.input) {
		nextChar, _ = utf8.DecodeRune(l.input[nextPos:])
	}
	if isLetter(nextChar) || isDigit(nextChar) || nextChar == '_' {
		return "", "", false
	}
	indent = l.text(l.pos, identifierPos)
	return indent, label, true
}

func (l *Lexer) nextHeredocToken() token.Token {
	if len(l.heredocTokens) == 0 {
		return token.Token{Type: token.T_ILLEGAL, Literal: "No heredoc tokens queued", Pos: token.Position{}}
	}
	tok := l.heredocTokens[0]
	l.heredocTokens = l.heredocTokens[1:]
	return tok
}
