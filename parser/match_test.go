package parser

import (
	"testing"

	"go-phpcs/ast"
	"go-phpcs/lexer"
)

func TestParseMatchExpressionBasic(t *testing.T) {
	code := `<?php
$result = match ($value) {
    1 => 'one',
    2 => 'two',
    default => 'other'
};`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	// Find the assignment statement
	assignStmt, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("Expected ExpressionStmt, got %T", nodes[0])
	}

	assign, ok := assignStmt.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("Expected AssignmentNode, got %T", assignStmt.Expr)
	}

	match, ok := assign.Right.(*ast.MatchNode)
	if !ok {
		t.Fatalf("Expected MatchNode, got %T", assign.Right)
	}

	// Check condition
	varNode, ok := match.Condition.(*ast.VariableNode)
	if !ok {
		t.Fatalf("Expected VariableNode condition, got %T", match.Condition)
	}
	if varNode.Name != "value" {
		t.Errorf("Expected condition variable 'value', got '%s'", varNode.Name)
	}

	// Check arms
	if len(match.Arms) != 3 {
		t.Fatalf("Expected 3 arms, got %d", len(match.Arms))
	}

	// Check first arm: 1 => 'one'
	arm1 := match.Arms[0]
	if len(arm1.Conditions) != 1 {
		t.Errorf("Expected 1 condition in first arm, got %d", len(arm1.Conditions))
	}
	intLit1, ok := arm1.Conditions[0].(*ast.IntegerNode)
	if !ok {
		t.Errorf("Expected IntegerNode condition, got %T", arm1.Conditions[0])
	}
	if intLit1.Value != 1 {
		t.Errorf("Expected condition value 1, got %d", intLit1.Value)
	}
	stringLit1, ok := arm1.Body.(*ast.StringLiteral)
	if !ok {
		t.Errorf("Expected StringLiteral body, got %T", arm1.Body)
	}
	if stringLit1.Value != "one" {
		t.Errorf("Expected body value 'one', got '%s'", stringLit1.Value)
	}

	// Check second arm: 2 => 'two'
	arm2 := match.Arms[1]
	intLit2, ok := arm2.Conditions[0].(*ast.IntegerNode)
	if !ok {
		t.Errorf("Expected IntegerNode condition, got %T", arm2.Conditions[0])
	}
	if intLit2.Value != 2 {
		t.Errorf("Expected condition value 2, got %d", intLit2.Value)
	}
	stringLit2, ok := arm2.Body.(*ast.StringLiteral)
	if !ok {
		t.Errorf("Expected StringLiteral body, got %T", arm2.Body)
	}
	if stringLit2.Value != "two" {
		t.Errorf("Expected body value 'two', got '%s'", stringLit2.Value)
	}

	// Check third arm: default => 'other'
	arm3 := match.Arms[2]
	defaultCond, ok := arm3.Conditions[0].(*ast.IdentifierNode)
	if !ok {
		t.Errorf("Expected IdentifierNode condition for default, got %T", arm3.Conditions[0])
	}
	if defaultCond.Value != "default" {
		t.Errorf("Expected condition value 'default', got '%s'", defaultCond.Value)
	}
	stringLit3, ok := arm3.Body.(*ast.StringLiteral)
	if !ok {
		t.Errorf("Expected StringLiteral body, got %T", arm3.Body)
	}
	if stringLit3.Value != "other" {
		t.Errorf("Expected body value 'other', got '%s'", stringLit3.Value)
	}
}

func TestParseMatchExpressionMultipleConditions(t *testing.T) {
	code := `<?php
$result = match ($value) {
    1, 2, 3 => 'small',
    4, 5 => 'medium',
    default => 'large'
};`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	// Find the match expression
	match := findMatchNode(t, nodes)
	if match == nil {
		t.Fatal("No match node found")
	}

	if len(match.Arms) != 3 {
		t.Fatalf("Expected 3 arms, got %d", len(match.Arms))
	}

	// Check first arm: 1, 2, 3 => 'small'
	arm1 := match.Arms[0]
	if len(arm1.Conditions) != 3 {
		t.Errorf("Expected 3 conditions in first arm, got %d", len(arm1.Conditions))
	}
	values := []int64{1, 2, 3}
	for i, cond := range arm1.Conditions {
		intLit, ok := cond.(*ast.IntegerNode)
		if !ok {
			t.Errorf("Expected IntegerNode condition, got %T", cond)
		}
		if intLit.Value != values[i] {
			t.Errorf("Expected condition value %d, got %d", values[i], intLit.Value)
		}
	}
	stringLit, ok := arm1.Body.(*ast.StringLiteral)
	if !ok {
		t.Errorf("Expected StringLiteral body, got %T", arm1.Body)
	}
	if stringLit.Value != "small" {
		t.Errorf("Expected body value 'small', got '%s'", stringLit.Value)
	}

	// Check second arm: 4, 5 => 'medium'
	arm2 := match.Arms[1]
	if len(arm2.Conditions) != 2 {
		t.Errorf("Expected 2 conditions in second arm, got %d", len(arm2.Conditions))
	}
}

func TestParseMatchExpressionComplexExpressions(t *testing.T) {
	code := `<?php
$result = match ($user->getType()) {
    User::ADMIN => $user->getAdminPanel(),
    User::MODERATOR => 'moderator',
    default => throw new Exception('Invalid user type')
};`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	match := findMatchNode(t, nodes)
	if match == nil {
		t.Fatal("No match node found")
	}

	// Check condition is a method call
	methodCall, ok := match.Condition.(*ast.MethodCallNode)
	if !ok {
		t.Fatalf("Expected MethodCallNode condition, got %T", match.Condition)
	}
	if methodCall.Method != "getType" {
		t.Errorf("Expected method 'getType', got '%s'", methodCall.Method)
	}

	if len(match.Arms) != 3 {
		t.Fatalf("Expected 3 arms, got %d", len(match.Arms))
	}

	// Check first arm has class constant condition
	arm1 := match.Arms[0]
	if len(arm1.Conditions) != 1 {
		t.Errorf("Expected 1 condition in first arm, got %d", len(arm1.Conditions))
	}

	// Check third arm has throw expression
	arm3 := match.Arms[2]
	throwNode, ok := arm3.Body.(*ast.ThrowNode)
	if !ok {
		t.Errorf("Expected ThrowNode body, got %T", arm3.Body)
	}
	if throwNode != nil {
		t.Log("Throw expression in match arm parsed successfully")
	}
}

func TestParseMatchExpressionTrailingCommas(t *testing.T) {
	code := `<?php
$result = match ($value) {
    1 => 'one',
    2 => 'two',
    default => 'other',
};`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	match := findMatchNode(t, nodes)
	if match == nil {
		t.Fatal("No match node found")
	}

	if len(match.Arms) != 3 {
		t.Fatalf("Expected 3 arms, got %d", len(match.Arms))
	}

	t.Log("Trailing comma in match expression parsed successfully")
}

func TestParseMatchExpressionEmpty(t *testing.T) {
	code := `<?php
$result = match ($value) {
};`

	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	match := findMatchNode(t, nodes)
	if match == nil {
		t.Fatal("No match node found")
	}

	if len(match.Arms) != 0 {
		t.Errorf("Expected 0 arms in empty match, got %d", len(match.Arms))
	}
}

func TestParseMatchExpressionErrors(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		errorMsg string
	}{
		{
			name:     "Missing opening parenthesis",
			code:     `<?php match $value { 1 => 'one' };`,
			errorMsg: "expected '(' after 'match'",
		},
		{
			name:     "Missing condition",
			code:     `<?php match () { 1 => 'one' };`,
			errorMsg: "expected condition expression in match",
		},
		{
			name:     "Missing closing parenthesis",
			code:     `<?php match ($value { 1 => 'one' };`,
			errorMsg: "expected ')' after match condition",
		},
		{
			name:     "Missing opening brace",
			code:     `<?php match ($value) 1 => 'one' };`,
			errorMsg: "expected '{' after match condition",
		},
		{
			name:     "Missing arrow",
			code:     `<?php match ($value) { 1 'one' };`,
			errorMsg: "expected '=>' after match conditions",
		},
		{
			name:     "Missing body expression",
			code:     `<?php match ($value) { 1 => };`,
			errorMsg: "expected expression after '=>' in match arm",
		},
		{
			name:     "Missing closing brace",
			code:     `<?php match ($value) { 1 => 'one' ;`,
			errorMsg: "expected '}' to close match expression",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.code)
			p := New(l, true)
			p.Parse()

			if len(p.Errors()) == 0 {
				t.Errorf("Expected parse error containing '%s', but got no errors", tc.errorMsg)
				return
			}

			if len(p.Errors()) == 0 {
				t.Errorf("Expected parse error for %s, but got no errors", tc.name)
			}
		})
	}
}

// Helper function to find match node in parsed nodes
func findMatchNode(t *testing.T, nodes []ast.Node) *ast.MatchNode {
	for _, node := range nodes {
		if exprStmt, ok := node.(*ast.ExpressionStmt); ok {
			if assign, ok := exprStmt.Expr.(*ast.AssignmentNode); ok {
				if match, ok := assign.Right.(*ast.MatchNode); ok {
					return match
				}
			}
		}
		// Also check for direct match expressions
		if exprStmt, ok := node.(*ast.ExpressionStmt); ok {
			if match, ok := exprStmt.Expr.(*ast.MatchNode); ok {
				return match
			}
		}
	}
	return nil
}
