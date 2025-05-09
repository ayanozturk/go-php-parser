package lexer

import (
	"go-phpcs/token"
	"strings"
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
	if l.char == '0' {
		switch l.peekChar() {
		case 'o', 'O':
			return l.readOctalNumber()
		case 'x', 'X':
			// Hexadecimal literal
			l.readChar() // consume '0'
			l.readChar() // consume 'x' or 'X'
			start := l.pos
			for (l.char >= '0' && l.char <= '9') || (l.char >= 'a' && l.char <= 'f') || (l.char >= 'A' && l.char <= 'F') || l.char == '_' {
				l.readChar()
			}
			// Remove underscores
			hex := strings.ReplaceAll(l.input[start:l.pos], "_", "")
			return "0x" + hex, false
		}
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
	// Implementation moved to lexer_symbol.go
	// This stub is left for reference; see lexer_symbol.go for helpers.
	switch l.char {
	case '+':
		return l.lexPlus(pos)
	case '-':
		return l.lexMinus(pos)
	case '*':
		return l.lexSingleChar(token.T_MULTIPLY, pos)
	case '/':
		return l.lexSlash(pos)
	case '|':
		return l.lexPipe(pos)
	case '>':
		return l.lexGreater(pos)
	case '<':
		return l.lexLess(pos)
	case '$':
		return l.lexDollar(pos)
	case '=':
		return l.lexEquals(pos)
	case '(': // ...single char tokens...
		return l.lexSingleChar(token.T_LPAREN, pos)
	case ')':
		return l.lexSingleChar(token.T_RPAREN, pos)
	case '{':
		return l.lexSingleChar(token.T_LBRACE, pos)
	case '}':
		return l.lexSingleChar(token.T_RBRACE, pos)
	case ';':
		return l.lexSingleChar(token.T_SEMICOLON, pos)
	case ',':
		return l.lexSingleChar(token.T_COMMA, pos)
	case '&':
		return l.lexSingleChar(token.T_AMPERSAND, pos)
	case '.':
		return l.lexDot(pos)
	case '"':
		return l.lexDoubleQuote(pos)
	case '\\':
		return l.lexBackslash(pos)
	case '\'':
		return l.lexSingleQuote(pos)
	case ':':
		return l.lexColon(pos)
	case '[':
		return l.lexSingleChar(token.T_LBRACKET, pos)
	case ']':
		return l.lexSingleChar(token.T_RBRACKET, pos)
	case '!':
		return l.lexSingleChar(token.T_NOT, pos)
	}
	return token.Token{Type: token.T_ILLEGAL, Literal: string(l.char), Pos: pos}
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
