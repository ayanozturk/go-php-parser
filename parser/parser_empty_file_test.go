package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestParseEmptyOrWhitespaceOnlyFile(t *testing.T) {
	for _, source := range []string{"", " \n\t"} {
		p := New(lexer.New(source), true)
		if nodes := p.Parse(); len(nodes) != 0 {
			t.Fatalf("expected no nodes for %q, got %#v", source, nodes)
		}
		if errs := p.Errors(); len(errs) != 0 {
			t.Fatalf("expected no parser errors for %q, got %v", source, errs)
		}
	}
}

func TestParseNonEmptyFileWithoutOpenTagStillErrors(t *testing.T) {
	p := New(lexer.New("not php"), true)
	_ = p.Parse()
	if errs := p.Errors(); len(errs) == 0 {
		t.Fatal("expected a missing PHP open-tag error")
	}
}
