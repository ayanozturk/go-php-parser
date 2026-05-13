package lexer

import (
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
	l := New(input)

	// Find the nowdoc token
	var nowdocToken token.Token
	for {
		tok := l.NextToken()
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

func TestLexerHeredoc(t *testing.T) {
	input := `<?php
$str = <<<EOT
This is a heredoc string!
With multiple lines.
And special chars: < > $ #
EOT;
$end = 1;`
	l := New(input)

	// Find the heredoc token
	var heredocToken token.Token
	for {
		tok := l.NextToken()
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

func TestLexerContinuesAfterHeredocTerminator(t *testing.T) {
	input := `<?php
$str = <<<EOT
hello
EOT;
$end = 1;`
	l := New(input)

	var seenSemicolon, seenEndVar bool
	for {
		tok := l.NextToken()
		if tok.Type == token.T_SEMICOLON {
			seenSemicolon = true
		}
		if tok.Type == token.T_VARIABLE && tok.Literal == "$end" {
			seenEndVar = true
			break
		}
		if tok.Type == token.T_EOF {
			break
		}
	}

	if !seenSemicolon {
		t.Fatal("expected semicolon token after heredoc terminator")
	}
	if !seenEndVar {
		t.Fatal("expected lexer to continue to the statement after heredoc")
	}
}

func TestLexerNowdocWithUnicodeContinuesAfterTerminator(t *testing.T) {
	input := "<?php\n$str = <<<'EOT'\ncheck ✓ and ✗\nEOT;\n$end = 1;"
	l := New(input)

	var seenSemicolon, seenEndVar bool
	for {
		tok := l.NextToken()
		if tok.Type == token.T_SEMICOLON {
			seenSemicolon = true
		}
		if tok.Type == token.T_VARIABLE && tok.Literal == "$end" {
			seenEndVar = true
			break
		}
		if tok.Type == token.T_EOF {
			break
		}
	}

	if !seenSemicolon {
		t.Fatal("expected semicolon token after unicode nowdoc terminator")
	}
	if !seenEndVar {
		t.Fatal("expected lexer to continue after unicode nowdoc content")
	}
}
