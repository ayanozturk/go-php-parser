package lexer

import (
	"github.com/ayanozturk/go-php-parser/token"
	"strings"
	"unicode/utf8"
)

func (l *Lexer) queueHeredocTokens(pos token.Position) {
	l.readChar() // consume first <
	l.readChar() // consume second <
	l.readChar() // consume third <
	l.skipWhitespace()

	identifier, isNowdoc := l.readHeredocIdentifier()
	if identifier == "" {
		l.heredocTokens = []token.Token{{Type: token.T_ILLEGAL, Literal: "Missing heredoc/nowdoc identifier", Pos: pos}}
		return
	}

	startType := token.T_START_HEREDOC
	if isNowdoc {
		startType = token.T_START_NOWDOC
	}
	startToken := token.Token{Type: startType, Literal: identifier, Pos: pos}

	l.skipToNextLine()

	body := l.readHeredocBody(identifier)
	bodyToken := token.Token{Type: token.T_ENCAPSED_AND_WHITESPACE, Literal: body, Pos: pos}

	endType := token.T_END_HEREDOC
	if isNowdoc {
		endType = token.T_END_NOWDOC
	}
	endToken := token.Token{Type: endType, Literal: identifier, Pos: pos}

	l.heredocTokens = []token.Token{startToken, bodyToken, endToken}
}

func (l *Lexer) readHeredocIdentifier() (string, bool) {
	if l.char == '\'' || l.char == '"' {
		quote := l.char
		l.readChar() // consume opening quote
		start := l.pos
		for l.char != quote && l.char != 0 {
			l.readChar()
		}
		identifier := l.input[start:l.pos]
		isNowdoc := (quote == '\'')
		if l.char == quote {
			l.readChar() // consume closing quote
		}
		return identifier, isNowdoc
	}
	start := l.pos
	for isLetter(l.char) || isDigit(l.char) || l.char == '_' {
		l.readChar()
	}
	identifier := l.input[start:l.pos]
	return identifier, false
}

func (l *Lexer) skipToNextLine() {
	for l.char != '\n' && l.char != 0 {
		l.readChar()
	}
	if l.char == '\n' {
		l.readChar()
	}
}

func (l *Lexer) readHeredocBody(identifier string) string {
	bodyStart := l.pos
	bodyEnd := -1
	terminatorIndent := ""
	// PHP identifiers are ASCII-only, so byte length == rune count.
	identByteLen := len(identifier)
	for l.char != 0 {
		lineStart := l.pos
		indent, ok := l.heredocTerminatorIndent(identifier)
		if ok {
			bodyEnd = lineStart
			terminatorIndent = indent
			// Advance past identifier using precomputed byte length.
			end := l.pos + len(indent) + identByteLen
			for l.pos < end {
				l.readChar()
			}
			break
		}
		l.skipToNextLine()
	}
	if bodyEnd == -1 {
		bodyEnd = l.pos
	}
	body := l.input[bodyStart:bodyEnd]
	if terminatorIndent != "" {
		body = dedentHeredocBody(body, terminatorIndent)
	}
	return body
}

func (l *Lexer) heredocTerminatorIndent(identifier string) (string, bool) {
	if identifier == "" {
		return "", false
	}
	identifierPos := l.pos
	for identifierPos < len(l.input) && (l.input[identifierPos] == ' ' || l.input[identifierPos] == '\t') {
		identifierPos++
	}
	if !strings.HasPrefix(l.input[identifierPos:], identifier) {
		return "", false
	}
	var nextChar rune
	nextPos := identifierPos + len(identifier)
	if nextPos < len(l.input) {
		nextChar, _ = utf8.DecodeRuneInString(l.input[nextPos:])
	} else {
		nextChar = 0
	}
	if isLetter(nextChar) || isDigit(nextChar) || nextChar == '_' {
		return "", false
	}
	return l.input[l.pos:identifierPos], true
}

func dedentHeredocBody(body, indent string) string {
	lines := strings.SplitAfter(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, indent) {
			lines[i] = strings.TrimPrefix(line, indent)
		}
	}
	return strings.Join(lines, "")
}

// nextHeredocToken emits the next queued heredoc token
func (l *Lexer) nextHeredocToken() token.Token {
	if len(l.heredocTokens) == 0 {
		return token.Token{Type: token.T_ILLEGAL, Literal: "No heredoc tokens queued", Pos: token.Position{}}
	}
	tok := l.heredocTokens[0]
	l.heredocTokens = l.heredocTokens[1:]
	return tok
}
