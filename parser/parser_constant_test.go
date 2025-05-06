package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func checkConstantNode(t *testing.T, node *ast.ConstantNode, wantName, wantValue, wantVisibility, wantType string) {
	t.Helper()
	if node == nil {
		t.Fatalf("parseConstant returned nil")
	}
	if node.Name != wantName {
		t.Errorf("expected name %s, got %s", wantName, node.Name)
	}
	if node.Value == nil || node.Value.TokenLiteral() != wantValue {
		t.Errorf("expected value %s, got %v", wantValue, node.Value)
	}
	if node.Visibility != wantVisibility {
		t.Errorf("expected visibility %s, got %s", wantVisibility, node.Visibility)
	}
	if node.Type != wantType {
		t.Errorf("expected type %s, got %s", wantType, node.Type)
	}
}

func parseConstantFromSource(src string, seekToken token.TokenType) *ast.ConstantNode {
	lex := lexer.New(src)
	p := New(lex, false)
	for p.tok.Type != seekToken && p.tok.Type != token.T_EOF {
		p.nextToken()
	}
	if p.tok.Type == token.T_EOF {
		return nil
	}
	return p.parseConstant()
}

func TestParseConstant(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		src := "<?php\nconst FOO = 123;"
		node := parseConstantFromSource(src, token.T_CONST)
		checkConstantNode(t, node, "FOO", "123", "", "")
	})

	t.Run("with visibility and type", func(t *testing.T) {
		src := "<?php\npublic const BAR: int = 42;"
		node := parseConstantFromSource(src, token.T_PUBLIC)
		checkConstantNode(t, node, "BAR", "42", "public", "int")
	})

	t.Run("with protected visibility", func(t *testing.T) {
		src := "<?php\nprotected const BAZ = 'hi';"
		node := parseConstantFromSource(src, token.T_PROTECTED)
		checkConstantNode(t, node, "BAZ", "hi", "protected", "")
	})
}
