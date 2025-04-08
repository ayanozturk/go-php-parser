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
			comment := l.readBlockComment()
			if strings.HasPrefix(comment, "/**") {
				return token.Token{Type: token.T_DOC_COMMENT, Literal: comment, Pos: pos}
			}
			return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
		} else {
			tok = token.Token{Type: token.T_DIVIDE, Literal: string(l.char), Pos: pos}
			l.readChar()
			return tok
		}
	case '<':
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
		if l.peekChar() == '>' {
			ch := l.char
			l.readChar()
			tok = token.Token{Type: token.T_DOUBLE_ARROW, Literal: string(ch) + string(l.char), Pos: pos}
		} else {
			tok = token.Token{Type: token.T_ASSIGN, Literal: string(l.char), Pos: pos}
		}
		l.readChar()
		return tok
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
	case '"':
		l.readChar() // consume opening quote
		str := l.readString('"')
		l.readChar() // consume closing quote
		return token.Token{Type: token.T_CONSTANT_ENCAPSED_STRING, Literal: str, Pos: pos}
	case '\'':
		l.readChar() // consume opening quote
		str := l.readString('\'')
		l.readChar() // consume closing quote
		return token.Token{Type: token.T_CONSTANT_STRING, Literal: str, Pos: pos}
	case ':':
		tok = token.Token{Type: token.T_COLON, Literal: string(l.char), Pos: pos}
		l.readChar()
		return tok
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
		Type:    token.TokenType(string(l.char)),
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

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
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
