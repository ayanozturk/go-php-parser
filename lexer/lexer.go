package lexer

import (
	"go-phpcs/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input    string
	pos      int
	readPos  int
	char     rune // Unicode-aware current character
	size     int  // Size of last rune read
	line     int
	column   int
	inString bool // Tracks if currently inside a string
	// For heredoc token queue
	heredocTokens []token.Token
}

// inStringMode returns whether the lexer is currently inside a string.
func (l *Lexer) inStringMode() bool {
	return l.inString
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

// readChar reads the next rune from input and advances position, supporting Unicode.
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.char = 0
		l.size = 0
	} else {
		l.char, l.size = utf8.DecodeRuneInString(l.input[l.readPos:])
	}
	l.pos = l.readPos
	l.readPos += l.size

	if l.char == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
}

// peekChar peeks the next rune without advancing position (Unicode-aware).
func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return r
}

func (l *Lexer) skipWhitespace() {
	for l.char == ' ' || l.char == '\t' || l.char == '\n' || l.char == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readString(quote byte) string {
	var out strings.Builder
	for l.char != rune(quote) && l.char != 0 {
		if l.char == '\\' {
			l.readChar()
			switch l.char {
			case 'n':
				out.WriteRune('\n')
			case 't':
				out.WriteRune('\t')
			case 'r':
				out.WriteRune('\r')
			case rune(quote):
				out.WriteRune(rune(quote))
			case '\\':
				out.WriteRune('\\')
			default:
				out.WriteRune('\\')
				out.WriteRune(l.char)
			}
		} else {
			out.WriteRune(l.char)
		}
		l.readChar()
	}
	return out.String()
}

// readOctalNumber reads and processes an octal number (0o format)
func (l *Lexer) readOctalNumber() (string, bool) {
	l.readChar() // consume '0'
	l.readChar() // consume 'o' or 'O'
	start := l.pos
	for (l.char >= '0' && l.char <= '7') || l.char == '_' {
		l.readChar()
	}
	// Remove underscores
	octal := strings.ReplaceAll(l.input[start:l.pos], "_", "")
	return "0o" + octal, false
}

func (l *Lexer) readNumber() (string, bool) {
	position := l.pos
	isFloat := false

	// PHP 8 octal literal: 0o or 0O
	if l.char == '0' && (l.peekChar() == 'o' || l.peekChar() == 'O') {
		return l.readOctalNumber()
	}

	for isDigit(l.char) || l.char == '.' || l.char == '_' {
		if l.char == '.' {
			if isFloat { // Second decimal point
				break
			}
			isFloat = true
		}
		if l.char == '_' {
			// PHP 7.4+ numeric literal separator, skip underscore
			l.readChar()
			continue
		}
		l.readChar()
	}

	// Remove underscores from the literal
	num := strings.ReplaceAll(l.input[position:l.pos], "_", "")
	return num, isFloat
}

// readIdentifier reads a PHP identifier (supports Unicode)
func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.char) || isDigit(l.char) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) NextToken() token.Token {
	if len(l.heredocTokens) > 0 {
		return l.nextHeredocToken()
	}
	l.skipWhitespace()
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}

	// Attributes
	if l.char == '#' && l.peekChar() == '[' {
		return l.lexAttribute(pos)
	}

	switch l.char {
	case '?':
		return l.lexQuestion(pos)
	case 0:
		return token.Token{Type: token.T_EOF, Literal: "", Pos: pos}
	case '+', '-', '*', '/', '|', '>', '<', '$', '=', '(', ')', '{', '}', ';', ',', '&', '.', '"', '\'', '\\', ':', '[', ']', '!':
		return l.lexSymbol(pos)
	}

	if isLetter(l.char) {
		return l.lexIdentifier(pos)
	}
	if isDigit(l.char) {
		return l.lexNumber(pos)
	}

	tok := token.Token{Type: token.T_ILLEGAL, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

// --- Helper methods for NextToken ---

func (l *Lexer) lexAttribute(pos token.Position) token.Token {
	startPos := l.pos
	startLine := l.line
	startCol := l.column
	l.readChar() // '#'
	l.readChar() // '['
	depth := 1
	for l.char != 0 && depth > 0 {
		if l.char == '[' {
			depth++
		} else if l.char == ']' {
			depth--
		}
		l.readChar()
	}
	endPos := l.pos
	attrLiteral := l.input[startPos:endPos]
	return token.Token{Type: token.T_ATTRIBUTE, Literal: attrLiteral, Pos: token.Position{Line: startLine, Column: startCol, Offset: startPos}}
}

func (l *Lexer) lexQuestion(pos token.Position) token.Token {
	if l.peekChar() == '-' && l.readPos+1 < len(l.input) && l.input[l.readPos+1] == '>' {
		l.readChar()
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_NULLSAFE_OBJECT_OPERATOR, Literal: "?->", Pos: pos}
	}
	if l.peekChar() == '?' {
		l.readChar()
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_COALESCE_EQUAL, Literal: "??=", Pos: pos}
		}
		l.readChar()
		return token.Token{Type: token.T_COALESCE, Literal: "??", Pos: pos}
	}
	tok := token.Token{Type: token.T_QUESTION, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexSymbol(pos token.Position) token.Token {
	switch l.char {
	case '+':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_PLUS_EQUAL, Literal: "+=", Pos: pos}
		}
		tok := token.Token{Type: token.T_PLUS, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_OBJECT_OPERATOR, Literal: "->", Pos: pos}
		}
		tok := token.Token{Type: token.T_MINUS, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '*':
		tok := token.Token{Type: token.T_MULTIPLY, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '/':
		return l.lexSlash(pos)
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_BOOLEAN_OR, Literal: "||", Pos: pos}
		}
		tok := token.Token{Type: token.T_PIPE, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_IS_GREATER_OR_EQUAL, Literal: ">=", Pos: pos}
		}
		tok := token.Token{Type: token.T_IS_GREATER, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '<':
		return l.lexLess(pos)
	case '$':
		l.readChar()
		if isLetter(l.char) {
			return token.Token{Type: token.T_VARIABLE, Literal: "$" + l.readIdentifier(), Pos: pos}
		}
	case '=':
		return l.lexEquals(pos)
	case '(':
		tok := token.Token{Type: token.T_LPAREN, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ')':
		tok := token.Token{Type: token.T_RPAREN, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '{':
		tok := token.Token{Type: token.T_LBRACE, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '}':
		tok := token.Token{Type: token.T_RBRACE, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ';':
		tok := token.Token{Type: token.T_SEMICOLON, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ',':
		tok := token.Token{Type: token.T_COMMA, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '&':
		tok := token.Token{Type: token.T_AMPERSAND, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '.':
		if l.peekChar() == '.' && l.input[l.readPos+1] == '.' {
			l.readChar()
			l.readChar()
			tok := token.Token{Type: token.T_ELLIPSIS, Literal: "...", Pos: pos}
			l.readChar()
			return tok
		}
		tok := token.Token{Type: token.T_DOT, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '"':
		l.inString = true
		l.readChar()
		str := l.readString('"')
		l.readChar()
		l.inString = false
		return token.Token{Type: token.T_CONSTANT_ENCAPSED_STRING, Literal: str, Pos: pos}
	case '\'':
		l.inString = true
		l.readChar()
		str := l.readString('\'')
		l.readChar()
		l.inString = false
		return token.Token{Type: token.T_CONSTANT_STRING, Literal: str, Pos: pos}
	case '\\':
		if l.inStringMode() {
			tok := token.Token{Type: token.T_BACKSLASH, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		}
		if isIdentifierStart(l.peekChar()) {
			tok := token.Token{Type: token.T_NS_SEPARATOR, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		}
		tok := token.Token{Type: token.T_BACKSLASH, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ':':
		if l.peekChar() == ':' {
			l.readChar()
			l.readChar()
			if l.peekChar() == 'c' && strings.HasPrefix(l.input[l.readPos:], "class") {
				l.readChar()
				l.readChar()
				l.readChar()
				l.readChar()
				l.readChar()
				return token.Token{Type: token.T_CLASS_CONST, Literal: "::class", Pos: pos}
			}
			return token.Token{Type: token.T_DOUBLE_COLON, Literal: "::", Pos: pos}
		}
		tok := token.Token{Type: token.T_COLON, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '[':
		tok := token.Token{Type: token.T_LBRACKET, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ']':
		tok := token.Token{Type: token.T_RBRACKET, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '!':
		tok := token.Token{Type: token.T_NOT, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	}
	return token.Token{Type: token.T_ILLEGAL, Literal: string(l.char), Pos: pos}
}

func (l *Lexer) lexSlash(pos token.Position) token.Token {
	if l.peekChar() == '/' {
		l.readChar()
		comment := l.readLineComment()
		return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
	} else if l.peekChar() == '*' {
		l.readChar()
		if l.peekChar() == '*' {
			l.readChar()
			comment := "/**" + l.readBlockComment()[2:]
			return token.Token{Type: token.T_DOC_COMMENT, Literal: comment, Pos: pos}
		}
		comment := l.readBlockComment()
		return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
	}
	tok := token.Token{Type: token.T_DIVIDE, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexLess(pos token.Position) token.Token {
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_IS_SMALLER_OR_EQUAL, Literal: "<=", Pos: pos}
	}
	if l.peekChar() == '<' && l.input[l.readPos+1] == '<' {
		l.queueHeredocTokens(pos)
		return l.nextHeredocToken()
	}
	if l.peekChar() == '?' {
		l.readChar()
		if l.peekChar() == 'p' {
			l.readChar()
			if l.peekChar() == 'h' {
				l.readChar()
				if l.peekChar() == 'p' {
					l.readChar()
					l.readChar()
					return token.Token{Type: token.T_OPEN_TAG, Literal: "<?php", Pos: pos}
				}
			}
		}
		tok := token.Token{Type: token.T_IS_SMALLER, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	}
	tok := token.Token{Type: token.T_IS_SMALLER, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexEquals(pos token.Position) token.Token {
	if l.peekChar() == '=' && l.input[l.readPos+1] == '=' {
		l.readChar()
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_IS_IDENTICAL, Literal: "===", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_IS_EQUAL, Literal: "==", Pos: pos}
	}
	if l.peekChar() == '>' {
		ch := l.char
		l.readChar()
		tok := token.Token{Type: token.T_DOUBLE_ARROW, Literal: string(ch) + string(l.char), Pos: pos}
		l.readChar()
		return tok
	}
	tok := token.Token{Type: token.T_ASSIGN, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexIdentifier(pos token.Position) token.Token {
	ident := l.readIdentifier()
	return LookupKeyword(ident, pos)
}

func (l *Lexer) lexNumber(pos token.Position) token.Token {
	num, isFloat := l.readNumber()
	if isFloat {
		return token.Token{Type: token.T_DNUMBER, Literal: num, Pos: pos}
	}
	return token.Token{Type: token.T_LNUMBER, Literal: num, Pos: pos}
}

func (l *Lexer) readLineComment() string {
	// Read line comment
	var out strings.Builder
	out.WriteRune('/')
	out.WriteRune('/')

	l.readChar() // Move past second /
	for l.char != '\n' && l.char != 0 {
		out.WriteRune(l.char)
		l.readChar()
	}

	return out.String()
}

func (l *Lexer) readBlockComment() string {
	// Read block comment
	var out strings.Builder
	out.WriteRune('/')
	out.WriteRune('*')

	l.readChar() // Move past *
	for {
		if l.char == 0 {
			break
		}

		if l.char == '*' && l.peekChar() == '/' {
			out.WriteRune('*')
			out.WriteRune('/')
			l.readChar() // consume *
			l.readChar() // consume /
			break
		}

		out.WriteRune(l.char)
		l.readChar()
	}

	return out.String()
}

// isLetter returns true if the byte/rune can start or be part of a PHP identifier (supports Unicode)
// isLetter returns true if the rune can start or be part of a PHP identifier (supports Unicode)
func isLetter(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.In(ch, unicode.Letter, unicode.Other_ID_Start)
}

// isIdentifierStart returns true if the byte can start a PHP identifier (namespace, variable, etc.)
// isIdentifierStart returns true if the rune can start a PHP identifier (namespace, variable, etc.)
func isIdentifierStart(ch rune) bool {
	return isLetter(ch)
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
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

// queueHeredocTokens scans and queues heredoc tokens
func (l *Lexer) queueHeredocTokens(pos token.Position) {
	l.readChar() // consume first <
	l.readChar() // consume second <
	l.readChar() // consume third <
	l.skipWhitespace()
	// Read heredoc/nowdoc identifier (quoted or unquoted)
	var identifier string
	var isNowdoc bool
	if l.char == '\'' || l.char == '"' {
		quote := l.char
		l.readChar() // consume opening quote
		start := l.pos
		for l.char != quote && l.char != 0 {
			l.readChar()
		}
		identifier = l.input[start:l.pos]
		isNowdoc = (quote == '\'')
		if l.char == quote {
			l.readChar() // consume closing quote
		}
	} else {
		start := l.pos
		for isLetter(l.char) || isDigit(l.char) || l.char == '_' {
			l.readChar()
		}
		identifier = l.input[start:l.pos]
		isNowdoc = false
	}
	if identifier == "" {
		l.heredocTokens = []token.Token{{Type: token.T_ILLEGAL, Literal: "Missing heredoc/nowdoc identifier", Pos: pos}}
		return
	}
	// Emit start heredoc/nowdoc token
	startType := token.T_START_HEREDOC
	if isNowdoc {
		startType = token.T_START_NOWDOC
	}
	startToken := token.Token{Type: startType, Literal: identifier, Pos: pos}
	// Skip to next line
	for l.char != '\n' && l.char != 0 {
		l.readChar()
	}
	if l.char == '\n' {
		l.readChar()
	}
	// Read heredoc/nowdoc body (Unicode-aware)
	identifierRunes := []rune(identifier)
	inputRunes := []rune(l.input)
	bodyStart := l.pos
	bodyEnd := -1
	for l.char != 0 {
		lineStart := l.pos
		// Check for ending identifier at start of line
		if l.char == identifierRunes[0] {
			match := true
			for i := 0; i < len(identifierRunes); i++ {
				if l.pos+i >= len(inputRunes) || inputRunes[l.pos+i] != identifierRunes[i] {
					match = false
					break
				}
			}
			if match {
				var nextChar rune
				if l.pos+len(identifierRunes) < len(inputRunes) {
					nextChar = inputRunes[l.pos+len(identifierRunes)]
				} else {
					nextChar = 0
				}
				if nextChar == '\n' || nextChar == ';' || nextChar == 0 {
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
			}
		}
		// Skip to next line
		for l.char != '\n' && l.char != 0 {
			l.readChar()
		}
		if l.char == '\n' {
			l.readChar()
		}
	}
	if bodyEnd == -1 {
		bodyEnd = l.pos
	}
	body := l.input[bodyStart:bodyEnd]
	bodyToken := token.Token{Type: token.T_ENCAPSED_AND_WHITESPACE, Literal: body, Pos: pos}
	endType := token.T_END_HEREDOC
	if isNowdoc {
		endType = token.T_END_NOWDOC
	}
	endToken := token.Token{Type: endType, Literal: identifier, Pos: pos}
	l.heredocTokens = []token.Token{startToken, bodyToken, endToken}
}

// PeekToken returns the next token without consuming it
func (l *Lexer) PeekToken() token.Token {
	pos := l.pos
	ch := l.char
	tok := l.NextToken()
	l.pos = pos
	l.char = ch
	return tok
}
