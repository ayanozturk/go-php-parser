package tests

import (
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func TestLexerHeredoc(t *testing.T) {
	input := `<?php
$str = <<<EOT
This is a heredoc string!
With multiple lines.
And special chars: < > $ #
EOT;
$end = 1;`
	lex := lexer.New(input)

	// Find the heredoc token
	var heredocToken token.Token
	for {
		tok := lex.NextToken()
		if tok.Type == token.T_ENCAPSED_AND_WHITESPACE {
			heredocToken = tok
			break
		}
		if tok.Type == token.T_EOF {
			t.Fatalf("Did not find heredoc token in input")
		}
	}

	expected := "This is a heredoc string!\nWith multiple lines.\nAnd special chars: < > $ #\n"
	if heredocToken.Literal != expected {
		t.Errorf("heredoc content mismatch.\nExpected: %q\nGot:      %q", expected, heredocToken.Literal)
	}
}
