package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseTernaryOperator(t *testing.T) {
	php := `<?php $result = 1 < 2 ? "yes" : "no";`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}
	stmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("Expected ExpressionStmt, got %T", nodes[0])
	}
	assign, ok := stmt.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("Expected AssignmentNode, got %T", stmt.Expr)
	}
	ternary, ok := assign.Right.(*ast.TernaryExpr)
	if !ok {
		t.Fatalf("Expected TernaryExpr on right side, got %T", assign.Right)
	}
	if ternary.Condition == nil || ternary.IfTrue == nil || ternary.IfFalse == nil {
		t.Error("TernaryExpr fields should not be nil")
	}
}

func TestParseShortTernaryOperator(t *testing.T) {
	php := `<?php $result = $this->users->first() ?: null;`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}
	stmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("Expected ExpressionStmt, got %T", nodes[0])
	}
	assign, ok := stmt.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("Expected AssignmentNode, got %T", stmt.Expr)
	}
	ternary, ok := assign.Right.(*ast.TernaryExpr)
	if !ok {
		t.Fatalf("Expected TernaryExpr on right side, got %T", assign.Right)
	}
	if ternary.Condition == nil || ternary.IfTrue == nil || ternary.IfFalse == nil {
		t.Fatal("Short ternary should populate condition, truthy branch, and falsey branch")
	}
	if ternary.IfTrue != ternary.Condition {
		t.Fatal("Short ternary should reuse the condition as the truthy branch")
	}
	if _, ok := ternary.IfFalse.(*ast.NullNode); !ok {
		t.Fatalf("Expected short ternary false branch to be null, got %T", ternary.IfFalse)
	}
	if _, ok := ternary.Condition.(*ast.MethodCallNode); !ok {
		t.Fatalf("Expected short ternary condition to remain the method call, got %T", ternary.Condition)
	}
}
