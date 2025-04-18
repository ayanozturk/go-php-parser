package tests

import (
	"testing"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
)

func TestParseArrayWithClassKey(t *testing.T) {
	input := `<?php
	return [
		Doctrine\Abc::class => ['all' => true],
	];`

	l := lexer.New(input)
	p := parser.New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}

	if len(nodes) > 0 {
		returnNode, ok := nodes[0].(*ast.ReturnNode)
		if !ok {
			t.Errorf("Expected ReturnNode, found %s", nodes[0].NodeType())
			return
		}

		arrayNode, ok := returnNode.Expr.(*ast.ArrayNode)
		if !ok {
			t.Errorf("Expected ArrayNode, found %s", returnNode.Expr.NodeType())
			return
		}

		if len(arrayNode.Elements) != 1 {
			t.Errorf("Expected ArrayNode to have 1 element, but got %d", len(arrayNode.Elements))
			return
		}
	}
}
