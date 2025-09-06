package parser

import (
	"testing"

	"go-phpcs/ast"
	"go-phpcs/lexer"
)

func TestMatchExpressionInFunctionCall(t *testing.T) {
	code := `<?php
$result = process(match ($status) {
    'active' => 1,
    'inactive' => 0,
    default => -1
});`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	// Find the assignment
	assign := findAssignment(t, nodes)
	funcCall, ok := assign.Right.(*ast.FunctionCallNode)
	if !ok {
		t.Fatalf("Expected FunctionCallNode, got %T", assign.Right)
	}

	nameNode, ok := funcCall.Name.(*ast.IdentifierNode)
	if !ok {
		t.Fatalf("Expected IdentifierNode for function name, got %T", funcCall.Name)
	}
	if nameNode.Value != "process" {
		t.Errorf("Expected function name 'process', got '%s'", nameNode.Value)
	}

	if len(funcCall.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(funcCall.Args))
	}

	match, ok := funcCall.Args[0].(*ast.MatchNode)
	if !ok {
		t.Fatalf("Expected MatchNode argument, got %T", funcCall.Args[0])
	}

	if len(match.Arms) != 3 {
		t.Errorf("Expected 3 arms, got %d", len(match.Arms))
	}

	t.Log("Match expression in function call parsed successfully")
}

func TestMatchExpressionInReturnStatement(t *testing.T) {
	code := `<?php
function getStatusCode($status) {
    return match ($status) {
        'success' => 200,
        'error' => 500,
        default => 404
    };
}`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	// Find the function
	funcNode := findFunction(t, nodes, "getStatusCode")
	returnStmt, ok := funcNode.Body[0].(*ast.ReturnNode)
	if !ok {
		t.Fatalf("Expected ReturnNode, got %T", funcNode.Body[0])
	}

	match, ok := returnStmt.Expr.(*ast.MatchNode)
	if !ok {
		t.Fatalf("Expected MatchNode in return, got %T", returnStmt.Expr)
	}

	if len(match.Arms) != 3 {
		t.Errorf("Expected 3 arms, got %d", len(match.Arms))
	}

	t.Log("Match expression in return statement parsed successfully")
}

func TestMatchExpressionInAssignment(t *testing.T) {
	code := `<?php
$value = match ($input) {
    1 => 'one',
    2 => 'two'
};`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	assign := findAssignment(t, nodes)
	match, ok := assign.Right.(*ast.MatchNode)
	if !ok {
		t.Fatalf("Expected MatchNode, got %T", assign.Right)
	}

	if len(match.Arms) != 2 {
		t.Errorf("Expected 2 arms, got %d", len(match.Arms))
	}

	t.Log("Match expression in assignment parsed successfully")
}

func TestMatchExpressionWithMethodCallCondition(t *testing.T) {
	code := `<?php
$result = match ($user->getRole()) {
    'admin' => 'Administrator',
    'user' => 'Regular User',
    default => 'Guest'
};`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	assign := findAssignment(t, nodes)
	match, ok := assign.Right.(*ast.MatchNode)
	if !ok {
		t.Fatalf("Expected MatchNode, got %T", assign.Right)
	}

	methodCall, ok := match.Condition.(*ast.MethodCallNode)
	if !ok {
		t.Fatalf("Expected MethodCallNode condition, got %T", match.Condition)
	}

	if methodCall.Method != "getRole" {
		t.Errorf("Expected method 'getRole', got '%s'", methodCall.Method)
	}

	t.Log("Match expression with method call condition parsed successfully")
}

// Helper functions

func findAssignment(t *testing.T, nodes []ast.Node) *ast.AssignmentNode {
	for _, node := range nodes {
		if exprStmt, ok := node.(*ast.ExpressionStmt); ok {
			if assign, ok := exprStmt.Expr.(*ast.AssignmentNode); ok {
				return assign
			}
		}
	}
	t.Fatal("No assignment found")
	return nil
}

func findFunction(t *testing.T, nodes []ast.Node, name string) *ast.FunctionNode {
	for _, node := range nodes {
		if funcNode, ok := node.(*ast.FunctionNode); ok {
			if funcNode.Name == name {
				return funcNode
			}
		}
	}
	t.Fatalf("Function %s not found", name)
	return nil
}
