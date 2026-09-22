package syntax

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestRecoveryFallbackScannersStopAtTopLevelDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		parse     func(*Parser) *GreenNode
		delimiter token.TokenType
	}{
		{"case separator", "junk({x}) :tail", (*Parser).parseUntilCaseSep, token.T_COLON},
		{"match arm comma", "junk({x}),tail", (*Parser).parseUntilMatchArmEnd, token.T_COMMA},
		{"comma or semicolon", "junk({x}),tail", (*Parser).parseUntilCommaOrSemi, token.T_COMMA},
		{"statement end", "junk({x});tail", (*Parser).parseUntilStmtEnd, token.T_SEMICOLON},
		{"parameter end", "junk((x)),tail", (*Parser).parseUntilParamEnd, token.T_COMMA},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser([]byte(tt.source))
			got := tt.parse(p)
			if got == nil || got.Kind() != KindTokenList {
				t.Fatalf("fallback node = %v, want token list", got)
			}
			if !p.at(tt.delimiter) {
				t.Fatalf("scanner stopped at token %v, want delimiter %v", p.tok().Type, tt.delimiter)
			}
			prefix := PrintGreenAt(got, []byte(tt.source), 0)
			if prefix == "" {
				t.Fatal("fallback scanner discarded the malformed input prefix")
			}
			if len(prefix) >= len(tt.source) {
				t.Fatalf("fallback scanner consumed delimiter and suffix: %q", prefix)
			}
		})
	}
}

func TestParseBalancedBlockTokenFallbackKeepsNestedInterpolation(t *testing.T) {
	source := []byte(`{ if ($x) { echo "{$x}"; } }tail`)
	p := NewParser(source)
	block := p.parseBalancedBlock()
	if block == nil || block.Kind() != KindTokenList {
		t.Fatalf("balanced block = %v, want token list", block)
	}
	if got := PrintGreenAt(block, source, 0); got != `{ if ($x) { echo "{$x}"; } }` {
		t.Fatalf("balanced block text = %q", got)
	}
	if p.tok().Literal != "tail" {
		t.Fatalf("next token = %q, want tail", p.tok().Literal)
	}
}

func TestPrecedingAttributePHPDocUsesClassSiblingContext(t *testing.T) {
	source := []byte(`<?php class C {
    /** property docs */
    #[A]
    public int $value;
    public int $plain;
}`)
	res := Parse(source)
	classLike := firstNodeOfKind(res.File.Root, KindClassDecl)
	var attributed, plain *RedNode
	Walk(classLike, func(n *RedNode) bool {
		if n.Kind() == KindPropertyDecl {
			if attributed == nil {
				attributed = copyRed(n)
			} else {
				plain = copyRed(n)
			}
		}
		return true
	})
	if attributed == nil || plain == nil {
		t.Fatalf("properties missing: attributed=%v plain=%v", attributed, plain)
	}
	doc := PrecedingAttributePHPDoc(classLike, attributed)
	if doc == nil || doc.RawContent == "" || doc.Description != "property docs" {
		t.Fatalf("preceding attribute PHPDoc = %v", doc)
	}
	if got := PrecedingAttributePHPDoc(classLike, plain); got != nil {
		t.Fatalf("plain property unexpectedly inherited PHPDoc %q", got.RawContent)
	}
	if PrecedingAttributePHPDoc(nil, attributed) != nil || PrecedingAttributePHPDoc(classLike, nil) != nil {
		t.Fatal("nil context should not produce a PHPDoc")
	}
}
