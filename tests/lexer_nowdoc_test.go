package tests

import (
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func TestLexerNowdoc(t *testing.T) {
	input := `<?php
$str = <<< 'EOHTML'
This is a nowdoc string!
No interpolation: $notAVar
EOHTML;
$end = 1;`
	lex := lexer.New(input)

	// Find the nowdoc token
	var nowdocToken token.Token
	for {
		tok := lex.NextToken()
		if tok.Type == token.T_ENCAPSED_AND_WHITESPACE {
			nowdocToken = tok
			break
		}
		if tok.Type == token.T_EOF {
			t.Fatalf("Did not find nowdoc token in input")
		}
	}

	expected := "This is a nowdoc string!\nNo interpolation: $notAVar\n"
	if nowdocToken.Literal != expected {
		t.Errorf("nowdoc content mismatch.\nExpected: %q\nGot:      %q", expected, nowdocToken.Literal)
	}
}
