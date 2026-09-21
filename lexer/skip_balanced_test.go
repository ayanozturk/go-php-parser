package lexer

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestSkipBalancedCurlyBlockHeredocCurly(t *testing.T) {
	// After pulling "{", depth starts at 1; heredoc {$label} must not close the body.
	src := []byte("{\n        $label = 'day';\n        $msg = <<<TXT\nlabel {$label}\nTXT;\n    }")
	l := NewBytes(src)
	tok := l.NextToken()
	if tok.Type != token.T_LBRACE {
		t.Fatalf("expected T_LBRACE, got %v", tok.Type)
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected balanced skip to succeed")
	}
	if end.Offset != len(src) {
		t.Fatalf("end offset %d, want %d", end.Offset, len(src))
	}
	if l.NextToken().Type != token.T_EOF {
		t.Fatal("expected EOF after body span")
	}
}

func TestSkipBalancedCurlyBlockDoubleQuotedCurly(t *testing.T) {
	src := []byte("{\n  $msg = \"label {$label}\";\n}")
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected balanced skip")
	}
	if end.Offset != len(src) {
		t.Fatalf("end %d want %d", end.Offset, len(src))
	}
}

func TestSkipBalancedCurlyBlockAttributeHash(t *testing.T) {
	src := []byte("{\n  #[Pure]\n  $x = 1;\n}")
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected balanced skip with #[Attr]")
	}
	if end.Offset != len(src) {
		t.Fatalf("end %d want %d", end.Offset, len(src))
	}
}
