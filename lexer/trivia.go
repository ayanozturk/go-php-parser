package lexer

import (
	"github.com/ayanozturk/go-php-parser/token"
)

// collectLeadingTrivia scans whitespace and comments and returns them as trivia
// tokens. Significant tokens keep these attached on LeadingTrivia so the
// parser never sees T_WHITESPACE / T_COMMENT / T_DOC_COMMENT as lookahead.
func (l *Lexer) collectLeadingTrivia() []token.Token {
	var trivia []token.Token
	for {
		if l.atEOF() {
			break
		}
		switch {
		case l.char == ' ' || l.char == '\t' || l.char == '\n' || l.char == '\r':
			trivia = append(trivia, l.lexWhitespaceTrivia())
		case l.char == '/' && (l.peekChar() == '/' || l.peekChar() == '*'):
			trivia = append(trivia, l.lexCommentTrivia())
		case l.char == '#' && l.peekChar() != '[':
			trivia = append(trivia, l.lexHashCommentTrivia())
		default:
			return trivia
		}
	}
	return trivia
}

func (l *Lexer) lexWhitespaceTrivia() token.Token {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	start := l.pos
	for l.char == ' ' || l.char == '\t' || l.char == '\n' || l.char == '\r' {
		l.readChar()
	}
	lit := l.text(start, l.pos)
	return token.Token{
		Type:    token.T_WHITESPACE,
		Literal: lit,
		Pos:     pos,
		End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
	}
}

func (l *Lexer) lexCommentTrivia() token.Token {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	commentStart := l.pos
	l.readChar() // '/'
	if l.char == '/' {
		lit := l.readLineComment(commentStart)
		return token.Token{
			Type:    token.T_COMMENT,
			Literal: lit,
			Pos:     pos,
			End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
		}
	}
	// block comment — l.char is '*'
	typ := token.T_COMMENT
	if l.peekChar() == '*' {
		// Zend: /**/ is a normal comment. Only promote to doc when the
		// second '*' is not immediately followed by the closer '/'.
		// Pre-consuming the second '*' before readBlockComment would leave
		// char on the closing '/' of /**/ and swallow the rest of the file.
		if l.readPos+1 >= len(l.input) || l.input[l.readPos+1] != '/' {
			l.readChar() // second '*'
			typ = token.T_DOC_COMMENT
		}
	}
	lit := l.readBlockComment(commentStart)
	// PHP treats /**/ as a normal comment, not a doc comment.
	if typ == token.T_DOC_COMMENT && lit == "/**/" {
		typ = token.T_COMMENT
	}
	return token.Token{
		Type:    typ,
		Literal: lit,
		Pos:     pos,
		End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
	}
}

func (l *Lexer) lexHashCommentTrivia() token.Token {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	lit := l.readHashComment()
	return token.Token{
		Type:    token.T_COMMENT,
		Literal: lit,
		Pos:     pos,
		End:     token.Position{Line: l.line, Column: l.column, Offset: l.pos},
	}
}
