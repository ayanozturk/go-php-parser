package lexer

import (
	"go-phpcs/token"
	"strings"
)

func (l *Lexer) lexPlus(pos token.Position) token.Token {
	if l.peekChar() == '+' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_INC, Literal: "++", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_PLUS_EQUAL, Literal: "+=", Pos: pos}
	}
	tok := token.Token{Type: token.T_PLUS, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexMinus(pos token.Position) token.Token {
	if l.peekChar() == '-' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_DEC, Literal: "--", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_MINUS_EQUAL, Literal: "-=", Pos: pos}
	}
	if l.peekChar() == '>' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_OBJECT_OPERATOR, Literal: "->", Pos: pos}
	}
	tok := token.Token{Type: token.T_MINUS, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexAsterisk(pos token.Position) token.Token {
	if l.peekChar() == '*' {
		l.readChar()
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_POW_EQUAL, Literal: "**=", Pos: pos}
		}
		l.readChar()
		return token.Token{Type: token.T_POW, Literal: "**", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_MUL_EQUAL, Literal: "*=", Pos: pos}
	}
	tok := token.Token{Type: token.T_MULTIPLY, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexPipe(pos token.Position) token.Token {
	if l.peekChar() == '|' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_BOOLEAN_OR, Literal: "||", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_OR_EQUAL, Literal: "|=", Pos: pos}
	}
	tok := token.Token{Type: token.T_PIPE, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexAmpersand(pos token.Position) token.Token {
	if l.peekChar() == '&' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_BOOLEAN_AND, Literal: "&&", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_AND_EQUAL, Literal: "&=", Pos: pos}
	}
	tok := token.Token{Type: token.T_AMPERSAND, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexCaret(pos token.Position) token.Token {
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_XOR_EQUAL, Literal: "^=", Pos: pos}
	}
	tok := token.Token{Type: token.T_CARET, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexGreater(pos token.Position) token.Token {
	if l.peekChar() == '>' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_SR, Literal: ">>", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_IS_GREATER_OR_EQUAL, Literal: ">=", Pos: pos}
	}
	tok := token.Token{Type: token.T_IS_GREATER, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexDollar(pos token.Position) token.Token {
	l.readChar()
	if isLetter(l.char) {
		for isLetter(l.char) || isDigit(l.char) {
			l.readChar()
		}
		return token.Token{Type: token.T_VARIABLE, Literal: l.input[pos.Offset:l.pos], Pos: pos}
	}
	return token.Token{Type: token.T_ILLEGAL, Literal: "$", Pos: pos}
}

func (l *Lexer) lexDot(pos token.Position) token.Token {
	if l.peekChar() == '.' && l.readPos+1 < len(l.input) && l.input[l.readPos+1] == '.' {
		l.readChar()
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_ELLIPSIS, Literal: "...", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_CONCAT_EQUAL, Literal: ".=", Pos: pos}
	}
	tok := token.Token{Type: token.T_DOT, Literal: asciiString(l.char), Pos: pos}
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
		tok := token.Token{Type: token.T_BACKSLASH, Literal: asciiString(l.char), Pos: pos}
		l.readChar()
		return tok
	}
	if isIdentifierStart(l.peekChar()) {
		tok := token.Token{Type: token.T_NS_SEPARATOR, Literal: asciiString(l.char), Pos: pos}
		l.readChar()
		return tok
	}
	tok := token.Token{Type: token.T_BACKSLASH, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexColon(pos token.Position) token.Token {
	if l.peekChar() == ':' {
		l.readChar()
		l.readChar()
		if l.peekChar() == 'c' && strings.HasPrefix(l.input[l.readPos:], "class") {
			afterClass := l.readPos + len("class")
			if afterClass < len(l.input) {
				next := rune(l.input[afterClass])
				if isLetter(next) || isDigit(next) {
					return token.Token{Type: token.T_DOUBLE_COLON, Literal: "::", Pos: pos}
				}
			}
			for i := 0; i < 5; i++ {
				l.readChar()
			}
			return token.Token{Type: token.T_CLASS_CONST, Literal: "::class", Pos: pos}
		}
		return token.Token{Type: token.T_DOUBLE_COLON, Literal: "::", Pos: pos}
	}
	tok := token.Token{Type: token.T_COLON, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexSingleChar(t token.TokenType, pos token.Position) token.Token {
	tok := token.Token{Type: t, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexSlash(pos token.Position) token.Token {
	if l.peekChar() == '/' {
		commentStart := l.pos // byte offset of first '/'
		l.readChar()          // now at second '/'
		comment := l.readLineComment(commentStart)
		return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
	} else if l.peekChar() == '*' {
		commentStart := l.pos // byte offset of opening '/'
		l.readChar()          // now at '*'
		if l.peekChar() == '*' {
			l.readChar() // now at second '*' (doc comment)
			comment := l.readBlockComment(commentStart)
			return token.Token{Type: token.T_DOC_COMMENT, Literal: comment, Pos: pos}
		}
		comment := l.readBlockComment(commentStart)
		return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
	}
	tok := token.Token{Type: token.T_DIVIDE, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexPercent(pos token.Position) token.Token {
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_MOD_EQUAL, Literal: "%=", Pos: pos}
	}
	tok := token.Token{Type: token.T_MODULO, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexLess(pos token.Position) token.Token {
	if l.peekChar() == '=' && l.readPos+1 < len(l.input) && l.input[l.readPos+1] == '>' {
		l.readChar()
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_SPACESHIP, Literal: "<=>", Pos: pos}
	}
	if l.peekChar() == '=' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_IS_SMALLER_OR_EQUAL, Literal: "<=", Pos: pos}
	}
	if l.peekChar() == '<' {
		if l.readPos+1 < len(l.input) && l.input[l.readPos+1] == '<' {
			l.queueHeredocTokens(pos)
			return l.nextHeredocToken()
		}
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_SL, Literal: "<<", Pos: pos}
	}
	if l.peekChar() == '?' {
		l.readChar()
		if l.peekChar() == 'p' || l.peekChar() == 'P' {
			l.readChar()
			if l.peekChar() == 'h' || l.peekChar() == 'H' {
				l.readChar()
				if l.peekChar() == 'p' || l.peekChar() == 'P' {
					l.readChar()
					l.readChar()
					return token.Token{Type: token.T_OPEN_TAG, Literal: "<?php", Pos: pos}
				}
			}
		}
		tok := token.Token{Type: token.T_IS_SMALLER, Literal: asciiString(l.char), Pos: pos}
		l.readChar()
		return tok
	}
	tok := token.Token{Type: token.T_IS_SMALLER, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexEquals(pos token.Position) token.Token {
	if l.peekChar() == '=' && l.readPos+1 < len(l.input) && l.input[l.readPos+1] == '=' {
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
		l.readChar()
		tok := token.Token{Type: token.T_DOUBLE_ARROW, Literal: "=>", Pos: pos}
		l.readChar()
		return tok
	}
	tok := token.Token{Type: token.T_ASSIGN, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexBang(pos token.Position) token.Token {
	if l.peekChar() == '=' {
		l.readChar()
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_IS_NOT_IDENTICAL, Literal: "!==", Pos: pos}
		}
		l.readChar()
		return token.Token{Type: token.T_IS_NOT_EQUAL, Literal: "!=", Pos: pos}
	}
	tok := token.Token{Type: token.T_NOT, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}
