package lexer

import (
	"go-phpcs/token"
	"strings"
	"unicode"
)

type Lexer struct {
	input        string
	pos          int
	readPos      int
	char         byte
	line         int
	column       int
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

func (l *Lexer) readString(quote byte) string {
	var out strings.Builder
	for l.char != quote && l.char != 0 {
		if l.char == '\\' {
			l.readChar()
			switch l.char {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			case quote:
				out.WriteByte(quote)
			case '\\':
				out.WriteByte('\\')
			default:
				out.WriteByte('\\')
				out.WriteByte(l.char)
			}
		} else {
			out.WriteByte(l.char)
		}
		l.readChar()
	}
	return out.String()
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
	// Prevent panic: ensure slice bounds are valid
	if position > len(l.input) {
		position = len(l.input)
	}
	if l.pos > len(l.input) {
		return l.input[position:]
	}
	return l.input[position:l.pos]
}

func (l *Lexer) NextToken() token.Token {
	// If we have heredoc tokens queued, emit them first
	if len(l.heredocTokens) > 0 {
		return l.nextHeredocToken()
	}
	var tok token.Token

	l.skipWhitespace()

	pos := token.Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.pos,
	}

	// PHP 8 attribute: #[...]
	if l.char == '#' && l.peekChar() == '[' {
		startPos := l.pos
		startLine := l.line
		startCol := l.column
		l.readChar() // consume '#'
		l.readChar() // consume '['
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
		return token.Token{
			Type:    token.T_ATTRIBUTE,
			Literal: attrLiteral,
			Pos:     token.Position{Line: startLine, Column: startCol, Offset: startPos},
		}
	}

	switch l.char {
	case 0:
		return token.Token{Type: token.T_EOF, Literal: "", Pos: pos}
	case '+':
		tok = token.Token{Type: token.T_PLUS, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '-':
		if l.peekChar() == '>' {
			l.readChar() // consume -
			l.readChar() // consume >
			return token.Token{Type: token.T_OBJECT_OPERATOR, Literal: "->", Pos: pos}
		}
		tok = token.Token{Type: token.T_MINUS, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '*':
		tok = token.Token{Type: token.T_MULTIPLY, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '/':
		if l.peekChar() == '/' {
			l.readChar() // consume second /
			comment := l.readLineComment()
			return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
		} else if l.peekChar() == '*' {
			l.readChar() // consume *
			if l.peekChar() == '*' {
				l.readChar()                                // consume second *
				comment := "/**" + l.readBlockComment()[2:] // include both asterisks
				return token.Token{Type: token.T_DOC_COMMENT, Literal: comment, Pos: pos}
			}
			comment := l.readBlockComment()
			return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
		} else {
			tok = token.Token{Type: token.T_DIVIDE, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		}
	case '?':
		if l.peekChar() == '?' {
			l.readChar() // consume first ?
			if l.peekChar() == '=' {
				l.readChar() // consume second ?
				l.readChar() // consume =
				return token.Token{Type: token.T_COALESCE_EQUAL, Literal: "??=", Pos: pos}
			} else {
				l.readChar() // consume second ?
				return token.Token{Type: token.T_COALESCE, Literal: "??", Pos: pos}
			}
		}
		tok = token.Token{Type: token.T_QUESTION, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '|':
		if l.peekChar() == '|' {
			l.readChar() // consume first |
			l.readChar() // consume second |
			return token.Token{Type: token.T_BOOLEAN_OR, Literal: "||", Pos: pos}
		}
		tok = token.Token{Type: token.T_PIPE, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '<':
		// Heredoc detection
		if l.peekChar() == '<' && l.input[l.readPos+1] == '<' {
			l.readChar() // consume first <
			l.readChar() // consume second <
			l.readChar() // consume third <
			l.skipWhitespace()
			// Read heredoc/nowdoc identifier (quoted or unquoted)
			var identifier string
			if l.char == '\'' || l.char == '"' {
				quote := l.char
				l.readChar() // consume opening quote
				start := l.pos
				for l.char != quote && l.char != 0 {
					l.readChar()
				}
				identifier = l.input[start:l.pos]
				if l.char == quote {
					l.readChar() // consume closing quote
				}
			} else {
				start := l.pos
				for isLetter(l.char) || isDigit(l.char) || l.char == '_' {
					l.readChar()
				}
				identifier = l.input[start:l.pos]
			}
			if identifier == "" {
				return token.Token{Type: token.T_ILLEGAL, Literal: "Missing heredoc/nowdoc identifier", Pos: pos}
			}
			// Skip to next line after identifier
			for l.char != '\n' && l.char != 0 {
				l.readChar()
			}
			if l.char == '\n' {
				l.readChar()
			}
			bodyStart := l.pos
			bodyEnd := -1
			for l.char != 0 {
				lineStart := l.pos
				// Only match ending identifier if it's at the start of a line
				if l.char == identifier[0] {
					match := true
					for i := 0; i < len(identifier); i++ {
						if l.input[l.pos+i] != identifier[i] {
							match = false
							break
						}
					}
					if match {
						nextChar := l.input[l.pos+len(identifier)]
						if nextChar == '\n' || nextChar == ';' || nextChar == 0 {
							bodyEnd = lineStart
							// Advance lexer past ending identifier
							for i := 0; i < len(identifier); i++ {
								l.readChar()
							}
							// If next char is semicolon, skip it
							if l.char == ';' {
								l.readChar()
							}
							// If next char is newline, skip it
							if l.char == '\n' {
								l.readChar()
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
			return token.Token{Type: token.T_ENCAPSED_AND_WHITESPACE, Literal: body, Pos: pos}
		}
		// Open tag detection
		if l.peekChar() == '?' {
			l.readChar() // consume ?
			if l.peekChar() == 'p' {
				l.readChar()
				if l.peekChar() == 'h' {
					l.readChar()
					if l.peekChar() == 'p' {
						l.readChar()
						l.readChar() // consume last char
						return token.Token{Type: token.T_OPEN_TAG, Literal: "<?php", Pos: pos}
					}
				}
			}
		}
	case '$':
		l.readChar()
		if isLetter(l.char) {
			tok = token.Token{
				Type:    token.T_VARIABLE,
				Literal: "$" + l.readIdentifier(),
				Pos:     pos,
			}
			return tok
		}
	case '=':
		// Longest match first: ===, ==, =>
		if l.peekChar() == '=' && l.input[l.readPos+1] == '=' {
			// ===
			l.readChar() // consume first =
			l.readChar() // consume second =
			l.readChar() // consume third =
			return token.Token{Type: token.T_IS_IDENTICAL, Literal: "===", Pos: pos}
		} else if l.peekChar() == '=' {
			// ==
			l.readChar() // consume first =
			l.readChar() // consume second =
			return token.Token{Type: token.T_IS_EQUAL, Literal: "==", Pos: pos}
		} else if l.peekChar() == '>' {
			// =>
			ch := l.char
			l.readChar()
			tok = token.Token{Type: token.T_DOUBLE_ARROW, Literal: string(ch) + string(l.char), Pos: pos}
			l.readChar()
			return tok
		} else {
			tok = token.Token{Type: token.T_ASSIGN, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		}
	case '(':
		tok = token.Token{Type: token.T_LPAREN, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ')':
		tok = token.Token{Type: token.T_RPAREN, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '{':
		tok = token.Token{Type: token.T_LBRACE, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '}':
		tok = token.Token{Type: token.T_RBRACE, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ';':
		tok = token.Token{Type: token.T_SEMICOLON, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ',':
		tok = token.Token{Type: token.T_COMMA, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '&':
		tok = token.Token{Type: token.T_AMPERSAND, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '.':
		if l.peekChar() == '.' && l.input[l.readPos+1] == '.' {
			l.readChar() // consume second .
			l.readChar() // consume third .
			tok = token.Token{Type: token.T_ELLIPSIS, Literal: "...", Pos: pos}
			l.readChar()
			return tok
		}
		// Emit dot operator for string concatenation
		tok = token.Token{Type: token.T_DOT, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case '"':
		l.inString = true // Enter string mode
		l.readChar() // consume opening quote
		str := l.readString('"')
		l.readChar() // consume closing quote
		l.inString = false // Exit string mode
		return token.Token{Type: token.T_CONSTANT_ENCAPSED_STRING, Literal: str, Pos: pos}
	case '\'':
		l.inString = true // Enter string mode
		l.readChar() // consume opening quote
		str := l.readString('\'')
		l.readChar() // consume closing quote
		l.inString = false // Exit string mode
		return token.Token{Type: token.T_CONSTANT_STRING, Literal: str, Pos: pos}
	case '\\':
		if l.inStringMode() {
			// Inside a string, treat as escape
			tok := token.Token{Type: token.T_BACKSLASH, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		} else if isIdentifierStart(l.peekChar()) {
			// In code, and next is identifier, treat as namespace separator
			tok := token.Token{Type: token.T_NS_SEPARATOR, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		} else {
			// Default fallback
			tok := token.Token{Type: token.T_BACKSLASH, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		}
	case ':':
		if l.peekChar() == ':' {
			l.readChar() // consume first :
			l.readChar() // consume second :

			// Check if "class" follows "::"
			if l.peekChar() == 'c' && strings.HasPrefix(l.input[l.readPos:], "class") {
				l.readChar() // consume 'c'
				l.readChar() // consume 'l'
				l.readChar() // consume 'a'
				l.readChar() // consume 's'
				l.readChar() // consume 's'
				return token.Token{Type: token.T_CLASS_CONST, Literal: "::class", Pos: pos}
			}

			return token.Token{Type: token.T_DOUBLE_COLON, Literal: "::", Pos: pos}
		} else {
			tok = token.Token{Type: token.T_COLON, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		}
	case '[':
		tok = token.Token{Type: token.T_LBRACKET, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	case ']':
		tok = token.Token{Type: token.T_RBRACKET, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
	}

	if isLetter(l.char) {
		ident := l.readIdentifier()

		// Check for keywords
		switch ident {
		case "function":
			return token.Token{Type: token.T_FUNCTION, Literal: ident, Pos: pos}
		case "if":
			return token.Token{Type: token.T_IF, Literal: ident, Pos: pos}
		case "else":
			return token.Token{Type: token.T_ELSE, Literal: ident, Pos: pos}
		case "elseif":
			return token.Token{Type: token.T_ELSEIF, Literal: ident, Pos: pos}
		case "endif":
			return token.Token{Type: token.T_ENDIF, Literal: ident, Pos: pos}
		case "array":
			return token.Token{Type: token.T_ARRAY, Literal: ident, Pos: pos}
		case "mixed":
			return token.Token{Type: token.T_MIXED, Literal: ident, Pos: pos}
		case "string":
			return token.Token{Type: token.T_STRING, Literal: ident, Pos: pos}
		case "callable":
			return token.Token{Type: token.T_CALLABLE, Literal: ident, Pos: pos}
		case "true":
			return token.Token{Type: token.T_TRUE, Literal: ident, Pos: pos}
		case "false":
			return token.Token{Type: token.T_FALSE, Literal: ident, Pos: pos}
		case "null":
			return token.Token{Type: token.T_NULL, Literal: ident, Pos: pos}
		case "class":
			return token.Token{Type: token.T_CLASS, Literal: ident, Pos: pos}
		case "extends":
			return token.Token{Type: token.T_EXTENDS, Literal: ident, Pos: pos}
		case "interface":
			return token.Token{Type: token.T_INTERFACE, Literal: ident, Pos: pos}
		case "instanceof":
			return token.Token{Type: token.T_INSTANCEOF, Literal: ident, Pos: pos}
		case "implements":
			return token.Token{Type: token.T_IMPLEMENTS, Literal: ident, Pos: pos}
		case "echo":
			return token.Token{Type: token.T_ECHO, Literal: ident, Pos: pos}
		case "new":
			return token.Token{Type: token.T_NEW, Literal: ident, Pos: pos}
		case "public":
			return token.Token{Type: token.T_PUBLIC, Literal: ident, Pos: pos}
		case "private":
			return token.Token{Type: token.T_PRIVATE, Literal: ident, Pos: pos}
		case "protected":
			return token.Token{Type: token.T_PROTECTED, Literal: ident, Pos: pos}
		case "return":
			return token.Token{Type: token.T_RETURN, Literal: ident, Pos: pos}
		case "enum":
			return token.Token{Type: token.T_ENUM, Literal: ident, Pos: pos}
		case "case":
			return token.Token{Type: token.T_CASE, Literal: ident, Pos: pos}
		case "trait":
			return token.Token{Type: token.T_TRAIT, Literal: ident, Pos: pos}
		default:
			return token.Token{Type: token.T_STRING, Literal: ident, Pos: pos}
		}
	}

	if isDigit(l.char) {
		num, isFloat := l.readNumber()
		if isFloat {
			return token.Token{Type: token.T_DNUMBER, Literal: num, Pos: pos}
		}
		return token.Token{Type: token.T_LNUMBER, Literal: num, Pos: pos}
	}

	tok = token.Token{
		Type:    token.T_ILLEGAL,
		Literal: string(l.char),
		Pos:     pos,
	}
	l.readChar()
	return tok
}

func (l *Lexer) readLineComment() string {
	var out strings.Builder
	out.WriteByte('/')
	out.WriteByte('/')

	l.readChar() // Move past second /
	for l.char != '\n' && l.char != 0 {
		out.WriteByte(l.char)
		l.readChar()
	}

	return out.String()
}

func (l *Lexer) readBlockComment() string {
	var out strings.Builder
	out.WriteByte('/')
	out.WriteByte('*')

	l.readChar() // Move past *
	for {
		if l.char == 0 {
			break
		}

		if l.char == '*' && l.peekChar() == '/' {
			out.WriteByte('*')
			out.WriteByte('/')
			l.readChar() // consume *
			l.readChar() // consume /
			break
		}

		out.WriteByte(l.char)
		l.readChar()
	}

	return out.String()
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

// isIdentifierStart returns true if the byte can start a PHP identifier (namespace, variable, etc.)
func isIdentifierStart(ch byte) bool {
	return isLetter(ch)
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
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
	// Read heredoc identifier
	start := l.pos
	for isLetter(l.char) || isDigit(l.char) || l.char == '_' {
		l.readChar()
	}
	identifier := l.input[start:l.pos]
	if identifier == "" {
		l.heredocTokens = []token.Token{{Type: token.T_ILLEGAL, Literal: "Missing heredoc identifier", Pos: pos}}
		return
	}
	// Emit start heredoc token
	startHeredocToken := token.Token{Type: token.T_START_HEREDOC, Literal: identifier, Pos: pos}
	// Skip to next line
	for l.char != '\n' && l.char != 0 {
		l.readChar()
	}
	if l.char == '\n' {
		l.readChar()
	}
	// Read heredoc body
	bodyStart := l.pos
	bodyEnd := -1
	for l.char != 0 {
		lineStart := l.pos
		// Check for ending identifier at start of line
		if l.char == identifier[0] {
			match := true
			for i := 0; i < len(identifier); i++ {
				if l.input[l.pos+i] != identifier[i] {
					match = false
					break
				}
			}
			if match {
				nextChar := l.input[l.pos+len(identifier)]
				if nextChar == '\n' || nextChar == ';' || nextChar == 0 {
					bodyEnd = lineStart
					l.pos += len(identifier)
					l.readPos = l.pos
					l.char = l.input[l.readPos-1]
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
	// Emit heredoc body as string (no interpolation)
	bodyToken := token.Token{Type: token.T_ENCAPSED_AND_WHITESPACE, Literal: body, Pos: pos}
	// Emit end heredoc token
	endHeredocToken := token.Token{Type: token.T_END_HEREDOC, Literal: identifier, Pos: pos}
	// Queue tokens
	l.heredocTokens = []token.Token{startHeredocToken, bodyToken, endHeredocToken}
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
