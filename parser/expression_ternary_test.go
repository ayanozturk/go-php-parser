package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
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

func TestBooleanAndDoesNotSwallowTernary(t *testing.T) {
	php := `<?php $n = is_string($x) && $x !== '' ? (int)$x : null;`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
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
		t.Fatalf("Expected TernaryExpr (PHP: && binds tighter than ?:), got %T", assign.Right)
	}
	cond, ok := ternary.Condition.(*ast.BinaryExpr)
	if !ok || cond.Operator != "&&" {
		t.Fatalf("Expected && condition, got %#v", ternary.Condition)
	}
	if _, ok := ternary.IfTrue.(*ast.TypeCastNode); !ok {
		t.Fatalf("Expected int cast in true branch, got %T", ternary.IfTrue)
	}
}

func TestBooleanAndOrAssociativity(t *testing.T) {
	php := `<?php $n = $a && $b || $c;`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	stmt := nodes[0].(*ast.ExpressionStmt)
	assign := stmt.Expr.(*ast.AssignmentNode)
	orExpr, ok := assign.Right.(*ast.BinaryExpr)
	if !ok || orExpr.Operator != "||" {
		t.Fatalf("Expected || at root (PHP: && tighter than ||), got %#v", assign.Right)
	}
	andExpr, ok := orExpr.Left.(*ast.BinaryExpr)
	if !ok || andExpr.Operator != "&&" {
		t.Fatalf("Expected && on left of ||, got %#v", orExpr.Left)
	}
}

func TestBooleanAndStillStealsAssignment(t *testing.T) {
	php := `<?php $n = $a && $b = $c;`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	stmt := nodes[0].(*ast.ExpressionStmt)
	assign := stmt.Expr.(*ast.AssignmentNode)
	andExpr, ok := assign.Right.(*ast.BinaryExpr)
	if !ok || andExpr.Operator != "&&" {
		t.Fatalf("Expected && at root, got %#v", assign.Right)
	}
	inner, ok := andExpr.Right.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("Expected assignment stolen into && right, got %T", andExpr.Right)
	}
	left, ok := inner.Left.(*ast.VariableNode)
	if !ok || left.Name != "b" {
		t.Fatalf("Expected $b assignment target, got %#v", inner.Left)
	}
}
