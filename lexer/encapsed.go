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

// commitConstantDoubleQuote scans a plain "..." once and, when there is no
// interpolation, advances the lexer and returns T_CONSTANT_ENCAPSED_STRING.
// When $ or {$ is found it restores the opening '"' and returns ok=false so
// the caller can take the encapsed path — avoiding the old look-then-
// retokenize double traversal of large constant strings.
func (l *Lexer) commitConstantDoubleQuote(pos token.Position) (token.Token, bool) {
	start := l.pos
	snapPos, snapReadPos, snapChar, snapSize := l.pos, l.readPos, l.char, l.size
	snapLine, snapCol := l.line, l.column

	l.readChar() // opening "
	for !l.atEOF() && l.char != '"' {
		if l.checkCancel() {
			lit := l.text(start, l.pos)
			return token.Token{
				Type:    token.T_CONSTANT_ENCAPSED_STRING,
				Literal: lit,
				Pos:     pos,
				End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
			}, true
		}
		if l.char == '\\' {
			l.readChar()
			if !l.atEOF() {
				l.readChar()
			}
			continue
		}
		if l.char == '$' || (l.char == '{' && l.peekChar() == '$') {
			l.pos, l.readPos, l.char, l.size = snapPos, snapReadPos, snapChar, snapSize
			l.line, l.column = snapLine, snapCol
			return token.Token{}, false
		}
		l.readChar()
	}
	if l.char == '"' {
		l.readChar()
	}
	lit := l.text(start, l.pos)
	return token.Token{
		Type:    token.T_CONSTANT_ENCAPSED_STRING,
		Literal: lit,
		Pos:     pos,
		End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
	}, true
}

// queueEncapsedStep emits the next encapsed/heredoc unit into the pending
// queue: one T_ENCAPSED_AND_WHITESPACE chunk, one interpolation burst
// ($var / $var[…] / ${…}), a terminator, or a {$ handoff. Callers drain via
// nextHeredocToken; scanToken invokes this once per NextToken so peak queue
// length stays bounded instead of materializing the whole body up front.
// nowdoc=false enables interpolation.
func (l *Lexer) queueEncapsedStep(nowdoc bool) {
	if l.atEOF() {
		l.encapsed = encapsedNone
		return
	}
	if l.checkCancel() {
		l.encapsed = encapsedNone
		return
	}
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
		if l.cancelErr != nil {
			l.encapsed = encapsedNone
		}
		return
	}
	if !nowdoc && l.char == '$' && (isLetter(l.peekChar()) || l.peekChar() == '{') {
		l.lexEncapsedVariable()
		// ${...} switches to normal mode until '}' (afterCurlyOpen).
		// Simple $var / $var[…] leave a short burst on the queue; scanToken
		// drains it before the next step.
		return
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
		l.braceExprDepth = 1
		l.encapsed = encapsedNone
		l.afterCurlyOpen = true
		return
	}
	l.lexEncapsedChunk(nowdoc)
	if l.cancelErr != nil {
		l.encapsed = encapsedNone
	}
}

// lexEncapsedChunk emits T_ENCAPSED_AND_WHITESPACE up to the next
// interpolation, terminator, or EOF. For nowdocs, runs to terminator only.
func (l *Lexer) lexEncapsedChunk(nowdoc bool) {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	start := l.pos
	for !l.atEOF() {
		if l.checkCancel() {
			break
		}
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
// expression that was started with {$ or ${. Restores encapsed mode only;
// the next scanToken step emits the following body token.
func (l *Lexer) resumeEncapsedAfterBrace() {
	if l.heredocLabel != "" {
		l.encapsed = encapsedHeredoc
	} else {
		l.encapsed = encapsedDoubleQuote
	}
	l.afterCurlyOpen = false
}
