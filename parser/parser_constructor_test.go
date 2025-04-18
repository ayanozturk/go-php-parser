package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseConstructorPromotionReadonly(t *testing.T) {
	input := `<?php
class Bar {
    public function __construct(
        public readonly string $foo,
        protected readonly ?Baz $baz = null,
        private $plain,
        $notPromoted = 123
    ) {}
}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, got none")
	}

	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}

	if len(classNode.Methods) == 0 {
		t.Fatalf("Expected at least one method in class")
	}

	ctor, ok := classNode.Methods[0].(*ast.FunctionNode)
	if !ok || ctor.Name != "__construct" {
		t.Fatalf("Expected constructor method, got %T, name=%v", classNode.Methods[0], ctor.Name)
	}

	params := ctor.Params
	if len(params) != 4 {
		t.Fatalf("Expected 4 parameters, got %d", len(params))
	}

	// public readonly string $foo
	p0 := params[0].(*ast.ParameterNode)
	if !p0.IsPromoted || p0.Visibility != "public" || p0.TypeHint != "string" || p0.Name != "foo" {
		t.Errorf("Param 0 not properly promoted: %+v", p0)
	}

	// protected readonly ?Baz $baz = null
	p1 := params[1].(*ast.ParameterNode)
	if !p1.IsPromoted || p1.Visibility != "protected" || p1.TypeHint != "?Baz" || p1.Name != "baz" {
		t.Errorf("Param 1 not properly promoted: %+v", p1)
	}

	// private $plain
	p2 := params[2].(*ast.ParameterNode)
	if !p2.IsPromoted || p2.Visibility != "private" || p2.Name != "plain" {
		t.Errorf("Param 2 not properly promoted: %+v", p2)
	}

	// $notPromoted = 123
	p3 := params[3].(*ast.ParameterNode)
	if p3.IsPromoted || p3.Visibility != "" || p3.Name != "notPromoted" {
		t.Errorf("Param 3 should not be promoted: %+v", p3)
	}
}

// TestParseConstructorPromotion_Original verifies legacy promotion parsing for regression coverage
func TestParseConstructorPromotion_Original(t *testing.T) {
	input := `<?php
class Foo {
    public function __construct(
        private array $tokens,
        protected ?Bar $bar = null,
        public $baz = 42,
        $notPromoted = "nope"
    ) {}
}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, got none")
	}

	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}

	if len(classNode.Methods) == 0 {
		t.Fatalf("Expected at least one method in class")
	}

	ctor, ok := classNode.Methods[0].(*ast.FunctionNode)
	if !ok || ctor.Name != "__construct" {
		t.Fatalf("Expected constructor method, got %T, name=%v", classNode.Methods[0], ctor.Name)
	}

	params := ctor.Params
	if len(params) != 4 {
		t.Fatalf("Expected 4 parameters, got %d", len(params))
	}

	// private array $tokens
	p0 := params[0].(*ast.ParameterNode)
	if !p0.IsPromoted || p0.Visibility != "private" || p0.TypeHint != "array" || p0.Name != "tokens" {
		t.Errorf("Param 0 not properly promoted: %+v", p0)
	}

	// protected ?Bar $bar = null
	p1 := params[1].(*ast.ParameterNode)
	if !p1.IsPromoted || p1.Visibility != "protected" || p1.TypeHint != "?Bar" || p1.Name != "bar" {
		t.Errorf("Param 1 not properly promoted: %+v", p1)
	}

	// public $baz = 42
	p2 := params[2].(*ast.ParameterNode)
	if !p2.IsPromoted || p2.Visibility != "public" || p2.Name != "baz" {
		t.Errorf("Param 2 not properly promoted: %+v", p2)
	}

	// $notPromoted = "nope"
	p3 := params[3].(*ast.ParameterNode)
	if p3.IsPromoted || p3.Visibility != "" || p3.Name != "notPromoted" {
		t.Errorf("Param 3 should not be promoted: %+v", p3)
	}
}
