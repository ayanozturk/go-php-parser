package lexer

import (
	"github.com/ayanozturk/go-php-parser/token"
)

// encapsedMode tracks interpolation scanning inside double-quoted strings
// and heredocs (nowdocs do not interpolate).
type encapsedMode int

const (
	encapsedNone encapsedMode = iota
	encapsedDoubleQuote
	encapsedHeredoc
)

// queueToken appends a finished token to the pending queue.
func (l *Lexer) queueToken(tok token.Token) {
	l.heredocTokens = append(l.heredocTokens, tok)
}

// lexAttribute emits only "#[", matching zend_language_scanner. Attribute
// contents are ordinary tokens parsed by the grammar.
func (l *Lexer) lexAttribute(pos token.Position) token.Token {
	l.readChar() // '#'
	l.readChar() // '['
	return token.Token{
		Type:    token.T_ATTRIBUTE,
		Literal: "#[",
		Pos:     pos,
		End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
	}
}

// lookDoubleQuoteConstant reports whether the double-quoted string starting at
// the current '"' has no interpolations and can be a single
// T_CONSTANT_ENCAPSED_STRING (including the quotes).
func (l *Lexer) lookDoubleQuoteConstant() bool {
	i := l.pos + 1
	for i < len(l.input) {
		c := l.input[i]
		switch c {
		case '\\':
			i++
			if i < len(l.input) {
				i++
			}
		case '"':
			return true
		case '$':
			return false
		case '{':
			if i+1 < len(l.input) && l.input[i+1] == '$' {
				return false
			}
			i++
		default:
			i++
		}
	}
	return true // unclosed — treat as constant scan path
}

// queueEncapsedBody tokenizes the interior of a double-quoted string or
// heredoc until the terminator. nowdoc=false enables interpolation.
func (l *Lexer) queueEncapsedBody(nowdoc bool) {
	for !l.atEOF() {
		if l.encapsed == encapsedDoubleQuote && l.char == '"' {
			pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
			l.readChar()
			l.queueToken(token.Token{
				Type:    token.T_CONSTANT_STRING,
				Literal: "\"",
				Pos:     pos,
				End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
			})
			l.encapsed = encapsedNone
			return
		}
		if l.encapsed == encapsedHeredoc {
			if indent, label, ok := l.matchHeredocTerminator(); ok {
				pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
				start := l.pos
				// consume indent + label
				endOff := l.pos + len(indent) + len(label)
				for l.pos < endOff {
					l.readChar()
				}
				endType := token.T_END_HEREDOC
				if nowdoc {
					endType = token.T_END_NOWDOC
				}
				l.queueToken(token.Token{
					Type:    endType,
					Literal: l.text(start, l.pos),
					Pos:     pos,
					End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
				})
				l.encapsed = encapsedNone
				l.heredocLabel = ""
				l.heredocNowdoc = false
				return
			}
		}
		if !nowdoc && l.char == '\\' {
			// escaped char stays in encapsed whitespace chunk
			l.lexEncapsedChunk(nowdoc)
			continue
		}
		if !nowdoc && l.char == '$' && (isLetter(l.peekChar()) || l.peekChar() == '{') {
			l.lexEncapsedVariable()
			// ${...} switches to normal mode until '}' (afterCurlyOpen), matching
			// the {$...} path which returns from this function. Continuing the
			// encapsed loop would swallow "}..." into T_ENCAPSED_AND_WHITESPACE.
			if l.afterCurlyOpen {
				return
			}
			continue
		}
		if !nowdoc && l.char == '{' && l.peekChar() == '$' {
			pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
			l.readChar() // '{'
			l.queueToken(token.Token{
				Type:    token.T_CURLY_OPEN,
				Literal: "{",
				Pos:     pos,
				End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
			})
			// Remaining "$var...}" is lexed by normal PHP mode until '}'.
			// Push a marker by temporarily leaving encapsed mode for the
			// braced expression; queue the closing brace when seen at depth 0.
			l.braceExprDepth = 1
			l.encapsed = encapsedNone
			l.afterCurlyOpen = true
			return
		}
		if !nowdoc && l.char == '$' && false {
			// handled above
		}
		l.lexEncapsedChunk(nowdoc)
	}
	l.encapsed = encapsedNone
}

// lexEncapsedChunk emits T_ENCAPSED_AND_WHITESPACE up to the next
// interpolation, terminator, or EOF. For nowdocs, runs to terminator only.
func (l *Lexer) lexEncapsedChunk(nowdoc bool) {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	start := l.pos
	for !l.atEOF() {
		if l.encapsed == encapsedDoubleQuote && l.char == '"' {
			break
		}
		if l.encapsed == encapsedHeredoc {
			if _, _, ok := l.matchHeredocTerminator(); ok {
				break
			}
		}
		if !nowdoc {
			if l.char == '\\' {
				l.readChar()
				if !l.atEOF() {
					l.readChar()
				}
				continue
			}
			if l.char == '$' {
				if isLetter(l.peekChar()) || l.peekChar() == '{' {
					break
				}
				l.readChar()
				continue
			}
			if l.char == '{' && l.peekChar() == '$' {
				break
			}
		}
		l.readChar()
	}
	if l.pos > start {
		tok := token.Token{
			Type:    token.T_ENCAPSED_AND_WHITESPACE,
			Literal: l.text(start, l.pos),
			Pos:     pos,
			End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
		}
		if l.encapsed != encapsedNone {
			l.queueToken(tok)
		} else {
			// called when already emitting — shouldn't happen
			l.queueToken(tok)
		}
	}
}

func (l *Lexer) lexEncapsedVariable() {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	if l.peekChar() == '{' {
		l.readChar() // $
		l.readChar() // {
		l.queueToken(token.Token{
			Type:    token.T_DOLLAR_OPEN_CURLY_BRACES,
			Literal: "${",
			Pos:     pos,
			End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
		})
		// varname inside ${...}
		if isLetter(l.char) {
			vpos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
			start := l.pos
			for isLetter(l.char) || isDigit(l.char) {
				l.readChar()
			}
			l.queueToken(token.Token{
				Type:    token.T_STRING_VARNAME,
				Literal: l.text(start, l.pos),
				Pos:     vpos,
				End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
			})
		}
		l.braceExprDepth = 1
		l.encapsed = encapsedNone
		l.afterCurlyOpen = true
		return
	}
	// simple $ident, optionally followed by [offset] which PHP emits as
	// separate tokens while still inside the string.
	l.readChar() // $
	if !isLetter(l.char) {
		// lone $ — include in encapsed text; back up isn't easy; emit as encapsed
		l.queueToken(token.Token{
			Type:    token.T_ENCAPSED_AND_WHITESPACE,
			Literal: "$",
			Pos:     pos,
			End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
		})
		return
	}
	for isLetter(l.char) || isDigit(l.char) {
		l.readChar()
	}
	l.queueToken(token.Token{
		Type:    token.T_VARIABLE,
		Literal: l.text(pos.Offset, l.pos),
		Pos:     pos,
		End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
	})
	// Simple string offset $var[0] / $var[foo]
	if l.char == '[' {
		bpos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
		l.readChar()
		l.queueToken(token.Token{
			Type:    token.T_LBRACKET,
			Literal: "[",
			Pos:     bpos,
			End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
		})
		if isDigit(l.char) {
			npos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
			start := l.pos
			for isDigit(l.char) {
				l.readChar()
			}
			l.queueToken(token.Token{
				Type:    token.T_NUM_STRING,
				Literal: l.text(start, l.pos),
				Pos:     npos,
				End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
			})
		} else if isLetter(l.char) {
			npos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
			start := l.pos
			for isLetter(l.char) || isDigit(l.char) {
				l.readChar()
			}
			l.queueToken(token.Token{
				Type:    token.T_STRING,
				Literal: l.text(start, l.pos),
				Pos:     npos,
				End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
			})
		} else if l.char == '$' {
			l.lexEncapsedVariable()
		}
		if l.char == ']' {
			epos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
			l.readChar()
			l.queueToken(token.Token{
				Type:    token.T_RBRACKET,
				Literal: "]",
				Pos:     epos,
				End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
			})
		}
	}
}

// resumeEncapsedAfterBrace is called when a '}' closes an interpolation
// expression that was started with {$ or ${.
func (l *Lexer) resumeEncapsedAfterBrace() {
	if l.heredocLabel != "" {
		l.encapsed = encapsedHeredoc
	} else {
		l.encapsed = encapsedDoubleQuote
	}
	l.afterCurlyOpen = false
	l.queueEncapsedBody(l.heredocNowdoc)
}
