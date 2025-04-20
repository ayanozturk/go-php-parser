package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestArrayVariables(t *testing.T) {
	input := `<?php
	$var3 = [1, 2, 3];
	`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}

	for _, node := range nodes {
		assignNode, ok := node.(*ast.AssignmentNode)
		if !ok {
			t.Errorf("Expected AssignmentNode, found %s", node.NodeType())
			continue
		}

		if _, ok := assignNode.Left.(*ast.VariableNode); !ok {
			t.Errorf("Expected Left to be VariableNode, found %s", assignNode.Left.NodeType())
		}

		arrayNode, ok := assignNode.Right.(*ast.ArrayNode)
		if !ok {
			t.Errorf("Expected Right to be ArrayNode, found %s", assignNode.Right.NodeType())
			continue
		}

		if len(arrayNode.Elements) != 3 {
			t.Errorf("Expected ArrayNode to have 3 elements, but got %d", len(arrayNode.Elements))
		}
	}
}
