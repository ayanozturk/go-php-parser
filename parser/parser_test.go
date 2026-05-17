package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParserInfiniteLoopScenarios(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "malformed if statement",
			input: `<?php
				if () {
					echo "test";
				}
			`,
			wantErr: true,
		},
		{
			name: "unclosed class body",
			input: `<?php
				class Test {
					public function test() {
					echo "test";
				// Missing closing braces
			`,
			wantErr: true,
		},
		{
			name: "malformed nested blocks",
			input: `<?php
				if (true) {
					if (false) {
						echo "test";
					// Missing closing brace for inner if
				}
			`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)
			_ = p.Parse()
			hasErr := len(p.Errors()) > 0
			if hasErr != tt.wantErr {
				t.Errorf("Test '%s': expected error=%v, got error=%v", tt.name, tt.wantErr, hasErr)
			}
		})
	}
}

func TestInstanceOf(t *testing.T) {
	l := lexer.New(`<?php
		if ($a instanceof \Exception) {
			echo "Exception";
		}
	`)
	p := New(l, true)
	_ = p.Parse()
	hasErr := len(p.Errors()) > 0
	if hasErr {
		t.Errorf("Test 'InstanceOf': expected no error, got error=%v", p.Errors())
	}
}

func TestIfConditionWithNotIdenticalAndBooleanAnd(t *testing.T) {
	l := lexer.New(`<?php
		if ('lint' !== $mode && false === getenv('SYMFONY_PATCH_TYPE_DECLARATIONS')) {
			echo 'ok';
		}
	`)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("expected no parser errors, got %v", p.Errors())
	}

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	ifNode, ok := nodes[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("expected IfNode, got %T", nodes[0])
	}

	cond, ok := ifNode.Condition.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr condition, got %T", ifNode.Condition)
	}

	if cond.Operator != "&&" {
		t.Fatalf("expected top-level && condition, got %q", cond.Operator)
	}

	left, ok := cond.Left.(*ast.BinaryExpr)
	if !ok || left.Operator != "!==" {
		t.Fatalf("expected left side !== comparison, got %T with operator %q", cond.Left, left.Operator)
	}

	right, ok := cond.Right.(*ast.BinaryExpr)
	if !ok || right.Operator != "===" {
		t.Fatalf("expected right side === comparison, got %T with operator %q", cond.Right, right.Operator)
	}
}

func TestInterfaceMethodParam(t *testing.T) {
	l := lexer.New(`<?php
interface AccessDecisionStrategyInterface
{
    public function decide(\Traversable $results): bool;
}
	`)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, got none")
	}
}
