package parser

import (
	"go-phpcs/lexer"
	"go-phpcs/token"
	"testing"
)

func TestParseConstant(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		src := "<?php\nconst FOO = 123;"
		lex := lexer.New(src)
		p := New(lex, false)
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
		if node.Visibility != "" {
			t.Errorf("expected no visibility, got %s", node.Visibility)
		}
		if node.Type != "" {
			t.Errorf("expected no type, got %s", node.Type)
		}
	})

	t.Run("with visibility and type", func(t *testing.T) {
		src := "<?php\npublic const BAR: int = 42;"
		lex := lexer.New(src)
		p := New(lex, false)
		for p.tok.Type != token.T_PUBLIC && p.tok.Type != token.T_EOF {
			p.nextToken()
		}
		node := p.parseConstant()
		if node == nil {
			t.Fatalf("parseConstant returned nil")
		}
		if node.Name != "BAR" {
			t.Errorf("expected name BAR, got %s", node.Name)
		}
		if node.Value == nil || node.Value.TokenLiteral() != "42" {
			t.Errorf("expected value 42, got %v", node.Value)
		}
		if node.Visibility != "public" {
			t.Errorf("expected visibility public, got %s", node.Visibility)
		}
		if node.Type != "int" {
			t.Errorf("expected type int, got %s", node.Type)
		}
	})

	t.Run("with protected visibility", func(t *testing.T) {
		src := "<?php\nprotected const BAZ = 'hi';"
		lex := lexer.New(src)
		p := New(lex, false)
		for p.tok.Type != token.T_PROTECTED && p.tok.Type != token.T_EOF {
			p.nextToken()
		}
		node := p.parseConstant()
		if node == nil {
			t.Fatalf("parseConstant returned nil")
		}
		if node.Name != "BAZ" {
			t.Errorf("expected name BAZ, got %s", node.Name)
		}
		if node.Value == nil || node.Value.TokenLiteral() != "hi" {
			t.Errorf("expected value hi, got %v", node.Value)
		}
		if node.Visibility != "protected" {
			t.Errorf("expected visibility protected, got %s", node.Visibility)
		}
		if node.Type != "" {
			t.Errorf("expected no type, got %s", node.Type)
		}
	})
}
