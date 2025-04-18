package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseSimpleEnum(t *testing.T) {
	input := `<?php
enum Status {
    case Pending;
    case Active;
    case Archived;
}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, got none")
	}

	enumNode, ok := nodes[0].(*ast.EnumNode)
	if !ok {
		t.Fatalf("Expected *ast.EnumNode, got %T", nodes[0])
	}

	if enumNode.Name != "Status" {
		t.Errorf("Expected enum name 'Status', got '%s'", enumNode.Name)
	}

	if len(enumNode.Cases) != 3 {
		t.Errorf("Expected 3 cases, got %d", len(enumNode.Cases))
	}

	expectedCases := []string{"Pending", "Active", "Archived"}
	for i, c := range enumNode.Cases {
		if c.Name != expectedCases[i] {
			t.Errorf("Expected case %d to be '%s', got '%s'", i, expectedCases[i], c.Name)
		}
	}
}
