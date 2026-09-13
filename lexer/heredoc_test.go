package lexer

import (
	"github.com/ayanozturk/go-php-parser/token"
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

	var body string
	var startLit string
	for {
		tok := l.NextToken()
		if tok.Type == token.T_START_NOWDOC {
			startLit = tok.Literal
		}
		if tok.Type == token.T_ENCAPSED_AND_WHITESPACE {
			body = tok.Literal
		}
		if tok.Type == token.T_EOF {
			break
		}
	}

	if !containsPrefix(startLit, "<<<") || !contains(startLit, "EOHTML") {
		t.Fatalf("T_START_NOWDOC literal should keep exact opener, got %q", startLit)
	}
	expected := "This is a nowdoc string!\nNo interpolation: $notAVar\n"
	if body != expected {
		t.Errorf("nowdoc content mismatch.\nExpected: %q\nGot:      %q", expected, body)
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

	var body string
	for {
		tok := l.NextToken()
		if tok.Type == token.T_ENCAPSED_AND_WHITESPACE {
			body += tok.Literal
		}
		if tok.Type == token.T_EOF {
			break
		}
	}

	expected := "This is a heredoc string!\nWith multiple lines.\nAnd special chars: < > $ #\n"
	if body != expected {
		t.Errorf("heredoc content mismatch.\nExpected: %q\nGot:      %q", expected, body)
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

func TestLexerIndentedNowdocPreservesBody(t *testing.T) {
	// Zend does not dedent in the lexer; indent is kept on body and end token.
	input := "<?php\n$sql = <<<'SQL'\n        SELECT 1\n    SQL;\n$end = 1;"
	l := New(input)

	var body, endLit string
	var seenSemicolon, seenEndVar bool
	for {
		tok := l.NextToken()
		switch {
		case tok.Type == token.T_ENCAPSED_AND_WHITESPACE:
			body = tok.Literal
		case tok.Type == token.T_END_NOWDOC:
			endLit = tok.Literal
		case tok.Type == token.T_SEMICOLON:
			seenSemicolon = true
		case tok.Type == token.T_VARIABLE && tok.Literal == "$end":
			seenEndVar = true
		}
		if seenEndVar || tok.Type == token.T_EOF {
			break
		}
	}

	if body != "        SELECT 1\n" {
		t.Fatalf("unexpected raw nowdoc body: %q", body)
	}
	if endLit != "    SQL" {
		t.Fatalf("T_END_NOWDOC should include indent, got %q", endLit)
	}
	if !seenSemicolon || !seenEndVar {
		t.Fatalf("expected lexer to continue after indented terminator; semicolon=%v end=%v", seenSemicolon, seenEndVar)
	}
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
