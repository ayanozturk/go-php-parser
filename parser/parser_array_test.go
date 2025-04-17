package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseArrayWithClassKey(t *testing.T) {
	input := `<?php
	return [
		Doctrine\Abc::class => ['all' => true],
	];`

	l := lexer.New(input)
	p := New(l, true)
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

		arrayItem, ok := arrayNode.Elements[0].(*ast.ArrayItemNode)
		if !ok {
			t.Errorf("Expected ArrayItemNode, found %s", arrayNode.Elements[0].NodeType())
			return
		}

		key, ok := arrayItem.Key.(*ast.IdentifierNode)
		if !ok {
			t.Errorf("Expected IdentifierNode as key, found %s", arrayItem.Key.NodeType())
			return
		}

		if key.Value != "Doctrine\\Abc::class" {
			t.Errorf("Expected key to be 'Doctrine\\Abc::class', but got '%s'", key.Value)
		}

		valueArray, ok := arrayItem.Value.(*ast.ArrayNode)
		if !ok {
			t.Errorf("Expected ArrayNode as value, found %s", arrayItem.Value.NodeType())
			return
		}

		if len(valueArray.Elements) != 1 {
			t.Errorf("Expected inner ArrayNode to have 1 element, but got %d", len(valueArray.Elements))
			return
		}

		innerItem, ok := valueArray.Elements[0].(*ast.ArrayItemNode)
		if !ok {
			t.Errorf("Expected ArrayItemNode in inner array, found %s", valueArray.Elements[0].NodeType())
			return
		}

		innerKey, ok := innerItem.Key.(*ast.StringNode)
		if !ok {
			t.Errorf("Expected StringNode as inner key, found %s", innerItem.Key.NodeType())
			return
		}

		if innerKey.Value != "all" {
			t.Errorf("Expected inner key to be 'all', but got '%s'", innerKey.Value)
		}

		innerValue, ok := innerItem.Value.(*ast.BooleanNode)
		if !ok {
			t.Errorf("Expected BooleanNode as inner value, found %s", innerItem.Value.NodeType())
			return
		}

		if !innerValue.Value {
			t.Errorf("Expected inner value to be true, but got false")
		}
	}
}
