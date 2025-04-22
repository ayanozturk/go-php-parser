package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseArrayAccessInIfCondition(t *testing.T) {
	input := `<?php
	if ($config['toolbar'] || $config['intercept_redirects'])
	{
	}`
	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}
	ifNode, ok := nodes[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("Expected IfNode, got %T", nodes[0])
	}
	// Check left side of || is an array access
	bin, ok := ifNode.Condition.(*ast.BinaryExpr)
	if !ok || bin.Operator != "||" {
		t.Fatalf("Expected BinaryExpr with ||, got %T", ifNode.Condition)
	}
	leftArr, ok := bin.Left.(*ast.ArrayAccessNode)
	if !ok {
		t.Errorf("Expected left side to be ArrayAccessNode, got %T", bin.Left)
	}
	if leftArr == nil || leftArr.Var == nil || leftArr.Index == nil {
		t.Errorf("ArrayAccessNode fields not set properly: %+v", leftArr)
	}
	// Check right side of || is also an array access
	rightArr, ok := bin.Right.(*ast.ArrayAccessNode)
	if !ok {
		t.Errorf("Expected right side to be ArrayAccessNode, got %T", bin.Right)
	}
	if rightArr == nil || rightArr.Var == nil || rightArr.Index == nil {
		t.Errorf("ArrayAccessNode fields not set properly: %+v", rightArr)
	}
}
