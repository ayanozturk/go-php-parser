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
		item, ok := arrayNode.Elements[0].(*ast.ArrayItemNode)
		if !ok {
			t.Errorf("Expected ArrayItemNode, got %T", arrayNode.Elements[0])
			return
		}
		// Check key is Doctrine\Abc::class
		key, ok := item.Key.(*ast.ClassConstFetchNode)
		if !ok {
			t.Errorf("Expected key to be ClassConstFetchNode, got %T", item.Key)
			return
		}
		if key.Class != "Doctrine\\Abc" || key.Const != "class" {
			t.Errorf("Expected key Doctrine\\Abc::class, got %s::%s", key.Class, key.Const)
		}
		// Check value is array ['all' => true]
		valArr, ok := item.Value.(*ast.ArrayNode)
		if !ok {
			t.Errorf("Expected value to be ArrayNode, got %T", item.Value)
			return
		}
		if len(valArr.Elements) != 1 {
			t.Errorf("Expected inner array to have 1 element, got %d", len(valArr.Elements))
			return
		}
		innerItem, ok := valArr.Elements[0].(*ast.ArrayItemNode)
		if !ok {
			t.Errorf("Expected inner element to be ArrayItemNode, got %T", valArr.Elements[0])
			return
		}
		keyStr, ok := innerItem.Key.(*ast.StringLiteral)
		if !ok || keyStr.Value != "all" {
			t.Errorf("Expected inner key to be string 'all', got %T (%v)", innerItem.Key, keyStr)
		}
		valBool, ok := innerItem.Value.(*ast.BooleanNode)
		if !ok || !valBool.Value {
			t.Errorf("Expected inner value to be boolean true, got %T (%v)", innerItem.Value, valBool)
		}
	}
}
