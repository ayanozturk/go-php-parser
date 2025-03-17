package lexer

import (
	"strings"

	"go-phpcs/token"
)

type Lexer struct {
	input string
	pos   int
}

func New(input string) *Lexer {
	return &Lexer{input: input}
}

func (l *Lexer) NextToken() token.Token {
	// Very simplified lexer for PoC
	input := l.input[l.pos:]
	if strings.HasPrefix(input, "<?php") {
		l.pos += 5
		return token.Token{Type: token.T_OPEN_TAG, Literal: "<?php"}
	}
	// Add more rules...
	return token.Token{Type: token.T_EOF, Literal: ""}
}
