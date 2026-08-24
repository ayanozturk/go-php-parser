package lexer

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

// TestNewFileStartsInHTMLModeWithoutLeadingOpenTag verifies that NewFile
// (used for real source files) treats content before the first "<?php"/"<?="
// tag as inline HTML, unlike New (used by bare-snippet tests), which always
// starts in PHP-code mode.
func TestNewFileStartsInHTMLModeWithoutLeadingOpenTag(t *testing.T) {
	input := "<div>Hello</div>\n<?php echo 1; ?>\nBye"
	l := NewFile(input)

	tok := l.NextToken()
	if tok.Type != token.T_INLINE_HTML {
		t.Fatalf("expected first token to be T_INLINE_HTML, got %s (%q)", tok.Type, tok.Literal)
	}
	if tok.Literal != "<div>Hello</div>\n" {
		t.Fatalf("unexpected inline HTML literal: %q", tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.T_OPEN_TAG {
		t.Fatalf("expected T_OPEN_TAG after inline HTML, got %s", tok.Type)
	}
}

func TestNewFileStartsInPHPModeWithLeadingOpenTag(t *testing.T) {
	l := NewFile("<?php echo 1;")
	tok := l.NextToken()
	if tok.Type != token.T_OPEN_TAG {
		t.Fatalf("expected first token to be T_OPEN_TAG, got %s", tok.Type)
	}
}

func TestNewFileStartsInPHPModeWithLeadingShortEchoTag(t *testing.T) {
	l := NewFile("<?= 1;")
	tok := l.NextToken()
	if tok.Type != token.T_OPEN_TAG {
		t.Fatalf("expected first token to be T_OPEN_TAG, got %s", tok.Type)
	}
}

// TestNewStillStartsInPHPModeForBareSnippets guards the existing behavior
// relied on by ~200 lexer/parser tests that construct bare code snippets
// (no leading "<?php") and expect immediate PHP tokenization.
func TestNewStillStartsInPHPModeForBareSnippets(t *testing.T) {
	l := New("$x = 1;")
	tok := l.NextToken()
	if tok.Type != token.T_VARIABLE {
		t.Fatalf("expected first token to be T_VARIABLE, got %s", tok.Type)
	}
}
