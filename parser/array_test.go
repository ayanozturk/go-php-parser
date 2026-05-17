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

func TestParseShortArrayLiteralElements(t *testing.T) {
	array := parseAssignedArray(t, `<?php $values = ['a' => 1, 'b' => 2];`)

	if len(array.Elements) != 2 {
		t.Fatalf("expected 2 array elements, got %d", len(array.Elements))
	}
	for i, element := range array.Elements {
		item, ok := element.(*ast.ArrayItemNode)
		if !ok {
			t.Fatalf("expected element %d to be ArrayItemNode, got %T", i, element)
		}
		if item.Key == nil {
			t.Fatalf("expected element %d to have a key", i)
		}
	}
}

func TestParseLongArrayLiteralWithCommentsAndTrailingComma(t *testing.T) {
	array := parseAssignedArray(t, `<?php
$values = array(
    'a' => 1,
    // intentionally disabled
    // 'b' => 2,
    'c' => 3,
);
`)

	if len(array.Elements) != 2 {
		t.Fatalf("expected comments to be skipped and 2 array elements parsed, got %d", len(array.Elements))
	}
}

func TestParseArrayElementFlags(t *testing.T) {
	array := parseAssignedArray(t, `<?php $values = [&$item, ...$items];`)

	if len(array.Elements) != 2 {
		t.Fatalf("expected 2 array elements, got %d", len(array.Elements))
	}
	byRef, ok := array.Elements[0].(*ast.ArrayItemNode)
	if !ok {
		t.Fatalf("expected first element to be ArrayItemNode, got %T", array.Elements[0])
	}
	if !byRef.ByRef || byRef.Unpack {
		t.Fatalf("expected first element to be by-reference only, got ByRef=%v Unpack=%v", byRef.ByRef, byRef.Unpack)
	}
	unpacked, ok := array.Elements[1].(*ast.ArrayItemNode)
	if !ok {
		t.Fatalf("expected second element to be ArrayItemNode, got %T", array.Elements[1])
	}
	if !unpacked.Unpack || unpacked.ByRef {
		t.Fatalf("expected second element to be unpacked only, got ByRef=%v Unpack=%v", unpacked.ByRef, unpacked.Unpack)
	}
}

func TestParseListLiteralWithSkippedElements(t *testing.T) {
	nodes := parsePHP(t, `<?php list($first, , $third) = $values;`)
	stmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("expected ExpressionStmt, got %T", nodes[0])
	}
	assign, ok := stmt.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("expected AssignmentNode, got %T", stmt.Expr)
	}
	list, ok := assign.Left.(*ast.ArrayNode)
	if !ok {
		t.Fatalf("expected list assignment target to be ArrayNode, got %T", assign.Left)
	}
	if len(list.Elements) != 2 {
		t.Fatalf("expected skipped list slot to be ignored and 2 elements parsed, got %d", len(list.Elements))
	}
}

func parseAssignedArray(t *testing.T, code string) *ast.ArrayNode {
	t.Helper()
	nodes := parsePHP(t, code)
	stmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("expected ExpressionStmt, got %T", nodes[0])
	}
	assign, ok := stmt.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("expected AssignmentNode, got %T", stmt.Expr)
	}
	array, ok := assign.Right.(*ast.ArrayNode)
	if !ok {
		t.Fatalf("expected assignment value to be ArrayNode, got %T", assign.Right)
	}
	return array
}

func parsePHP(t *testing.T, code string) []ast.Node {
	t.Helper()
	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}
	return nodes
}
