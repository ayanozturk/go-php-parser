package lexer

import (
	"go-phpcs/token"
	"strings"
)

func (l *Lexer) lexPlus(pos token.Position) token.Token {
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_PLUS_EQUAL, Literal: "+=", Pos: pos}
	}
	tok := token.Token{Type: token.T_PLUS, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexMinus(pos token.Position) token.Token {
	if l.peekChar() == '>' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_OBJECT_OPERATOR, Literal: "->", Pos: pos}
	}
	tok := token.Token{Type: token.T_MINUS, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexPipe(pos token.Position) token.Token {
	if l.peekChar() == '|' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_BOOLEAN_OR, Literal: "||", Pos: pos}
	}
	tok := token.Token{Type: token.T_PIPE, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexGreater(pos token.Position) token.Token {
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_IS_GREATER_OR_EQUAL, Literal: ">=", Pos: pos}
	}
	tok := token.Token{Type: token.T_IS_GREATER, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexDollar(pos token.Position) token.Token {
	l.readChar()
	if isLetter(l.char) {
		return token.Token{Type: token.T_VARIABLE, Literal: "$" + l.readIdentifier(), Pos: pos}
	}
	return token.Token{Type: token.T_ILLEGAL, Literal: "$", Pos: pos}
}

func (l *Lexer) lexDot(pos token.Position) token.Token {
	if l.peekChar() == '.' && l.input[l.readPos+1] == '.' {
		l.readChar()
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_ELLIPSIS, Literal: "...", Pos: pos}
	}
	tok := token.Token{Type: token.T_DOT, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexDoubleQuote(pos token.Position) token.Token {
	l.inString = true
	l.readChar()
	str := l.readString('"')
	l.readChar()
	l.inString = false
	return token.Token{Type: token.T_CONSTANT_ENCAPSED_STRING, Literal: str, Pos: pos}
}

func (l *Lexer) lexSingleQuote(pos token.Position) token.Token {
	l.inString = true
	l.readChar()
	str := l.readString('\'')
	l.readChar()
	l.inString = false
	return token.Token{Type: token.T_CONSTANT_STRING, Literal: str, Pos: pos}
}

func (l *Lexer) lexBackslash(pos token.Position) token.Token {
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
}

func (l *Lexer) lexColon(pos token.Position) token.Token {
	if l.peekChar() == ':' {
		l.readChar()
		l.readChar()
		if l.peekChar() == 'c' && strings.HasPrefix(l.input[l.readPos:], "class") {
			for i := 0; i < 5; i++ {
				l.readChar()
			}
			return token.Token{Type: token.T_CLASS_CONST, Literal: "::class", Pos: pos}
		}
		return token.Token{Type: token.T_DOUBLE_COLON, Literal: "::", Pos: pos}
	}
	tok := token.Token{Type: token.T_COLON, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexSingleChar(t token.TokenType, pos token.Position) token.Token {
	tok := token.Token{Type: t, Literal: string(l.char), Pos: pos}
	l.readChar()
	return tok
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
