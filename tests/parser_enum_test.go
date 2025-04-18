package tests

import (
	"testing"

	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
)

func TestParseEnum(t *testing.T) {
	code := `<?php
enum Status { case CASE_ONE; case CASE_TWO; }`
	lex := lexer.New(code)
	p := parser.New(lex, true)
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
