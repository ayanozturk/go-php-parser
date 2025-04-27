package lexer

import (
	"go-phpcs/token"
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
	identifierRunes := []rune(identifier)
	inputRunes := []rune(l.input)
	bodyStart := l.pos
	bodyEnd := -1
	for l.char != 0 {
		lineStart := l.pos
		if l.isEndOfHeredocLine(identifierRunes, inputRunes) {
			bodyEnd = lineStart
			l.pos += len(identifierRunes)
			l.readPos = l.pos
			if l.pos > 0 && l.pos-1 < len(inputRunes) {
				l.char = inputRunes[l.pos-1]
			} else {
				l.char = 0
			}
			break
		}
		l.skipToNextLine()
	}
	if bodyEnd == -1 {
		bodyEnd = l.pos
	}
	return l.input[bodyStart:bodyEnd]
}

func (l *Lexer) isEndOfHeredocLine(identifierRunes, inputRunes []rune) bool {
	if l.char != identifierRunes[0] {
		return false
	}
	for i := 0; i < len(identifierRunes); i++ {
		if l.pos+i >= len(inputRunes) || inputRunes[l.pos+i] != identifierRunes[i] {
			return false
		}
	}
	var nextChar rune
	if l.pos+len(identifierRunes) < len(inputRunes) {
		nextChar = inputRunes[l.pos+len(identifierRunes)]
	} else {
		nextChar = 0
	}
	return nextChar == '\n' || nextChar == ';' || nextChar == 0
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
