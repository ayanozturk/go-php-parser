package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestConstrutor(t *testing.T) {
	input := `<?php
	final class OptimizerExtension extends AbstractExtension
{
    public function __construct(
        private int $optimizers = -1,
    ) {
    }
}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		filtered := 0
		for _, err := range p.Errors() {
			if err != "empty union type" {
				filtered++
			}
		}
		if filtered > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
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
	if len(params) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(params))
	}
}

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
		filtered := 0
		for _, err := range p.Errors() {
			if err != "empty union type" {
				filtered++
			}
		}
		if filtered > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
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
}

func TestParseClassWithMultipleModifiers(t *testing.T) {
	php := `<?php
class Foo {
    public int $a = 1;
    protected static string $b;
    private function bar() {}
    public function baz() {}
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}
	// Optionally, check for ClassNode with correct methods and properties
}

func TestParseClassErrorRecovery_TypedProperties(t *testing.T) {
	php := `<?php
class Broken {
    public int $a = 1;
    public function foo() {}
    public string $b = 2;
    public function bar() {}
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}
	if len(classNode.Properties) != 2 {
		t.Fatalf("Expected 2 properties, got %d", len(classNode.Properties))
	}
	if len(classNode.Methods) != 2 {
		t.Fatalf("Expected 2 methods, got %d", len(classNode.Methods))
	}
}

func TestParseFunctionWithStaticReturnType(t *testing.T) {
	php := `<?php
class Foo {
    public function bar(): static {
        return new static();
    }
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}
	if len(classNode.Methods) == 0 {
		t.Fatal("Expected at least one method in class")
	}
	method, ok := classNode.Methods[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("Expected FunctionNode for method, got %T", classNode.Methods[0])
	}
	if method.ReturnType != "static" {
		t.Errorf("Expected return type 'static', got %q", method.ReturnType)
	}
}
