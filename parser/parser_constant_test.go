package parser

import (
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func TestParseConstant(t *testing.T) {
	src := "<?php\nconst FOO = 123;"
	lex := lexer.New(src)
	p := New(lex, false)
	// Advance to T_CONST
	for p.tok.Type != token.T_CONST && p.tok.Type != token.T_EOF {
		p.nextToken()
	}
	if p.tok.Type != token.T_CONST {
		t.Fatalf("did not find T_CONST token")
	}
	node := p.parseConstant()
	if node == nil {
		t.Fatalf("parseConstant returned nil")
	}
	if node.Name != "FOO" {
		t.Errorf("expected name FOO, got %s", node.Name)
	}
	if node.Value == nil || node.Value.TokenLiteral() != "123" {
		t.Errorf("expected value 123, got %v", node.Value)
	}
}
