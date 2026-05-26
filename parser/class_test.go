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

func TestSkipFunctionBodiesKeepsFollowingClassMembers(t *testing.T) {
	input := `<?php
class Example {
    public function first(): void {
        if (true) {
            echo "}";
        }
    }

    public function second(int $value): string {
        return (string) $value;
    }

    private string $name;
}`

	l := lexer.New(input)
	p := New(l, true)
	p.SkipFunctionBodies = true
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected one class node, got %d", len(nodes))
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}
	if len(classNode.Methods) != 2 {
		t.Fatalf("Expected two methods after skipping bodies, got %d", len(classNode.Methods))
	}
	if len(classNode.Properties) != 1 {
		t.Fatalf("Expected one property after skipping bodies, got %d", len(classNode.Properties))
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

func TestParseClassWithFQCNExtendsAndImplements(t *testing.T) {
	php := `<?php
class TemplateDirIterator extends \IteratorIterator implements \Stringable, Countable {}
`

	l := lexer.New(php)
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
	if classNode.Extends != "\\IteratorIterator" {
		t.Fatalf("Expected extends \\IteratorIterator, got %q", classNode.Extends)
	}
	if len(classNode.Implements) != 2 {
		t.Fatalf("Expected 2 implemented interfaces, got %d", len(classNode.Implements))
	}
	if classNode.Implements[0] != "\\Stringable" {
		t.Fatalf("Expected first interface \\Stringable, got %q", classNode.Implements[0])
	}
	if classNode.Implements[1] != "Countable" {
		t.Fatalf("Expected second interface Countable, got %q", classNode.Implements[1])
	}
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

func TestParseClassPropertyHooks(t *testing.T) {
	php := `<?php
class BackedProperty
{
    public private(set) string $name {
        get => $this->name;
        set => $value;
    }
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}
	if len(classNode.Properties) != 1 {
		t.Fatalf("Expected 1 property, got %d", len(classNode.Properties))
	}
	prop, ok := classNode.Properties[0].(*ast.PropertyNode)
	if !ok {
		t.Fatalf("Expected PropertyNode, got %T", classNode.Properties[0])
	}
	if prop.Visibility != "public" {
		t.Fatalf("Expected public visibility, got %q", prop.Visibility)
	}
	if prop.SetVisibility != "private" {
		t.Fatalf("Expected private set visibility, got %q", prop.SetVisibility)
	}
	if prop.TypeHint != "string" {
		t.Fatalf("Expected string type hint, got %q", prop.TypeHint)
	}
	if len(prop.Hooks) != 2 {
		t.Fatalf("Expected 2 property hooks, got %d", len(prop.Hooks))
	}
	if prop.Hooks[0].Name != "get" || prop.Hooks[0].Expr == nil {
		t.Fatalf("Expected get hook with expression, got %+v", prop.Hooks[0])
	}
	if prop.Hooks[1].Name != "set" || prop.Hooks[1].Expr == nil {
		t.Fatalf("Expected set hook with expression, got %+v", prop.Hooks[1])
	}
}

func TestParseClassAsymmetricVisibilityProperty(t *testing.T) {
	php := `<?php
class VisibilityFixture
{
    private(set) string $type;
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}
	prop, ok := classNode.Properties[0].(*ast.PropertyNode)
	if !ok {
		t.Fatalf("Expected PropertyNode, got %T", classNode.Properties[0])
	}
	if prop.SetVisibility != "private" {
		t.Fatalf("Expected private set visibility, got %q", prop.SetVisibility)
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

func TestParsePHPDocInClass(t *testing.T) {
	input := `<?php
/**
 * Interface for visiting nodes
 * @author Test Author
 */
class TestClass {
    public function visit(): void {}
}`

	l := lexer.New(input)
	p := New(l, false)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("No nodes parsed")
	}

	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}

	// Check class PHPDoc
	if classNode.PHPDoc == nil {
		t.Error("Expected PHPDoc for class, got nil")
	} else {
		if classNode.PHPDoc.Description != "Interface for visiting nodes" {
			t.Errorf("Expected class description 'Interface for visiting nodes', got %q", classNode.PHPDoc.Description)
		}
	}
}
