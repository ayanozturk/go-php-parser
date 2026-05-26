package parser

import (
	"testing"

	"go-phpcs/ast"
	"go-phpcs/lexer"
)

func TestParseEnum(t *testing.T) {
	code := `<?php
enum Status { case CASE_ONE; case CASE_TWO; }`
	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}

	enumNode, ok := nodes[0].(*ast.EnumNode)
	if !ok {
		t.Fatalf("Expected *ast.EnumNode, got %T", nodes[0])
	}

	if enumNode.Name != "Status" {
		t.Errorf("Expected enum name 'Status', got '%s'", enumNode.Name)
	}

	if len(enumNode.Cases) != 2 {
		t.Errorf("Expected 2 cases, got %d", len(enumNode.Cases))
	}
	if enumNode.Cases[0].Name != "CASE_ONE" {
		t.Errorf("Expected first case 'CASE_ONE', got '%s'", enumNode.Cases[0].Name)
	}
	if enumNode.Cases[1].Name != "CASE_TWO" {
		t.Errorf("Expected second case 'CASE_TWO', got '%s'", enumNode.Cases[1].Name)
	}
}

func TestParseEnumWithMethod(t *testing.T) {
	code := `<?php
enum ExpressionParserType: string {
    case Prefix = 'prefix';
    case Infix = 'infix';

    public static function getType(object $object): ExpressionParserType
    {
        return self::Prefix;
    }
}`
	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}
	enumNode, ok := nodes[0].(*ast.EnumNode)
	if !ok {
		t.Fatalf("Expected *ast.EnumNode, got %T", nodes[0])
	}
	if len(enumNode.Cases) != 2 {
		t.Fatalf("Expected 2 cases, got %d", len(enumNode.Cases))
	}
	if len(enumNode.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(enumNode.Methods))
	}
}

func TestParseEnumImplements(t *testing.T) {
	php := `<?php
enum Status: string implements \Serializable, JsonSerializable {
    case Active = 'active';
}`
	nodes := parsePHP(t, php)
	enumNode, ok := nodes[0].(*ast.EnumNode)
	if !ok {
		t.Fatalf("Expected *ast.EnumNode, got %T", nodes[0])
	}
	if enumNode.BackedBy != "string" {
		t.Fatalf("expected string backing type, got %q", enumNode.BackedBy)
	}
	if len(enumNode.Implements) != 2 {
		t.Fatalf("expected 2 implemented interfaces, got %d", len(enumNode.Implements))
	}
	if enumNode.Implements[0] != "\\Serializable" || enumNode.Implements[1] != "JsonSerializable" {
		t.Fatalf("unexpected enum implements: %#v", enumNode.Implements)
	}
}
