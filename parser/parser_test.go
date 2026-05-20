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
		{
			name: "malformed function call arguments with invalid operator",
			input: `<?php
				foo( + );
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

func TestClassWithTrailingComment(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "class with phpcs:ignore comment before brace",
			input: `<?php
class Authentication_Controller extends Public_Controller // phpcs:ignore
{
    public function login() {
        return true;
    }
}
`,
		},
		{
			name: "class with comment after extends",
			input: `<?php
class Foo extends Bar // some comment
{
    public function test() {}
}
`,
		},
		{
			name: "class with comment after implements",
			input: `<?php
class Foo implements Bar // phpcs:ignore
{
    public function test() {}
}
`,
		},
		{
			name: "class with comment between extends and implements",
			input: `<?php
class Foo extends Bar // comment
    implements Baz
{
    public function test() {}
}
`,
		},
		{
			name: "abstract class with trailing comment",
			input: `<?php
abstract class Foo // phpcs:ignore
{
    abstract public function test();
}
`,
		},
		{
			name: "trait with trailing comment",
			input: `<?php
trait MyTrait // phpcs:ignore
{
    public function test() {}
}
`,
		},
		{
			name: "enum with trailing comment",
			input: `<?php
enum Status // phpcs:ignore
{
    case Active;
    case Inactive;
}
`,
		},
		{
			name: "backed enum with trailing comment before brace",
			input: `<?php
enum Color: string // phpcs:ignore
{
    case Red = 'red';
}
`,
		},
		{
			name: "namespace with trailing comment before brace",
			input: `<?php
namespace App\Controllers // phpcs:ignore
{
    class Foo {}
}
`,
		},
		{
			name: "namespace inline with trailing comment before semicolon",
			input: `<?php
namespace App\Controllers; // phpcs:ignore
class Foo {}
`,
		},
		{
			name: "if block with trailing semicolon followed by another if",
			input: `<?php
class Foo {
    public function test() {
        if ($x) {
            echo "hi";
        };
        if ($y) {
            echo "bye";
        }
    }
}
`,
		},
		{
			name: "empty statement inside if block",
			input: `<?php
class Foo {
    public function test() {
        if ($x === false) {
            $a = parent::getUser()->getName();
            ;
        }
        echo "after";
    }
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l, true)
			nodes := p.Parse()

			if len(p.Errors()) > 0 {
				t.Errorf("expected no parser errors, got %v", p.Errors())
			}

			if len(nodes) == 0 {
				t.Errorf("expected at least one node, got none")
			}

			// Verify class nodes are properly parsed (not BlockNodes)
			for _, node := range nodes {
				if _, ok := node.(*ast.BlockNode); ok {
					t.Errorf("got unexpected BlockNode - class body was not parsed correctly")
				}
			}
		})
	}
}
