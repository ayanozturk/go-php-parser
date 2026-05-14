package lexer

import "strings"

func (l *Lexer) readLineComment() string {
	var out strings.Builder
	out.WriteRune('/')
	out.WriteRune('/')

	l.readChar() // move past second /
	for l.char != '\n' && l.char != 0 {
		out.WriteRune(l.char)
		l.readChar()
	}

	return out.String()
}

func (l *Lexer) readHashComment() string {
	var out strings.Builder
	out.WriteRune('#')

	l.readChar() // move past #
	for l.char != '\n' && l.char != 0 {
		out.WriteRune(l.char)
		l.readChar()
	}

	return out.String()
}

func (l *Lexer) readBlockComment() string {
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
