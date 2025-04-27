package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseFunctionWithUnionAndNamedParameters(t *testing.T) {
	input := `<?php
	function edgeCase(mixed|null $mixed, string $string) {
		return $mixed . $string;
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

	if len(nodes) > 0 {
		fn, ok := nodes[0].(*ast.FunctionNode)
		if !ok || len(fn.Params) == 0 {
			t.Errorf("Expected FunctionNode with parameters")
			return
		}
		param0, ok0 := fn.Params[0].(*ast.ParamNode)
		param1, ok1 := fn.Params[1].(*ast.ParamNode)
		if !ok0 || !ok1 {
			t.Errorf("Expected parameters to be *ast.ParamNode, got %T and %T", fn.Params[0], fn.Params[1])
			return
		}
		if param0.Name != "mixed" {
			t.Errorf("Expected parameter 1 name to be 'mixed', but got '%s'", param0.Name)
		}
		if param0.TypeHint != "mixed|null" {
			t.Errorf("Expected parameter 1 type hint to be 'mixed|null', but got '%s'", param0.TypeHint)
		}
		if param1.Name != "string" {
			t.Errorf("Expected parameter 2 name to be 'string', but got '%s'", param1.Name)
		}
		if param1.TypeHint != "string" {
			t.Errorf("Expected parameter 2 type hint to be 'string', but got '%s'", param1.TypeHint)
		}
	}
}

func TestParseFunctionWithCallable(t *testing.T) {
	input := `<?php
	function edgeCase(callable $callable) {
		return $callable();
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

	if len(nodes) > 0 {
		fn, ok := nodes[0].(*ast.FunctionNode)
		if !ok || len(fn.Params) == 0 {
			t.Errorf("Expected FunctionNode with parameters")
			return
		}
		param0, ok0 := fn.Params[0].(*ast.ParamNode)
		if !ok0 {
			t.Errorf("Expected parameter to be *ast.ParamNode, got %T", fn.Params[0])
			return
		}
		if param0.Name != "callable" {
			t.Errorf("Expected parameter 1 name to be 'callable', but got '%s'", param0.Name)
		}
		if param0.TypeHint != "callable" {
			t.Errorf("Expected parameter 1 type hint to be 'callable', but got '%s'", param0.TypeHint)
		}
	}
}

func TestParseStaticMethodInClass(t *testing.T) {
	input := `<?php
class Foo {
    public static function bar() {}
}`
	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, but got none")
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
	foundStatic := false
	for _, m := range method.Modifiers {
		if m == "static" {
			foundStatic = true
		}
	}
	if !foundStatic {
		t.Errorf("Expected 'static' in method.Modifiers, got %+v", method.Modifiers)
	}
}
func TestParseFunctionWithUnionTypeParam(t *testing.T) {
	input := `<?php
function foo(\DOMException|\Dom\Exception $e, array $a, Stub $stub, bool $isNested) {}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}
}

func TestParseFunctionWithAttributeAndUnionReturnType(t *testing.T) {
	input := `<?php
#[SomeAttr]
function foo(string $a): int|string
{
    return 1;
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
}

func TestParseFunctionWithComplexUnionAndNullableTypes(t *testing.T) {
	input := `<?php
function bar(array|string|null $x = null, ?string $y = "abc"): array|string|false {}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}
}

func TestParseFunctionWithParenthesizedTypeParam(t *testing.T) {
	input := `<?php
class X {
    public function setParent((NodeDefinition&ParentNodeDefinitionInterface)|null $parent): static {}
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
}

func TestParseTernaryWithFuncNumArgs(t *testing.T) {
	php := `<?php
class Foo {
    public function bar($a) {
        $nbToken = 1 < \func_num_args() ? func_get_arg(1) : 1;
    }

    public function baz(): void {
        if (\func_num_args() > 2) {
            $eventSourceOptions = func_get_arg(2);
        }
    }
}
`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}
	// Optionally, check for the expected structure (class, method, assignment, ternary, function calls)
}

func TestParseThrowStatementAndExpression(t *testing.T) {
	php := `<?php
function foo() {
    throw new \Exception("fail");
}
function bar($x) {
    $y = $x ?? throw new \Exception("fail");
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) < 2 {
		t.Fatal("Expected at least two function nodes")
	}
	// Optionally, check for ThrowNode in AST structure
}

func TestParseNullCoalescingAssignment(t *testing.T) {
	php := `<?php $a ??= 1; $this->foo ??= 2;` // property fetch
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) < 2 {
		t.Fatal("Expected at least two statements")
	}
	// Optionally, check for AssignmentNode with Operator "??="
}

func TestParseFunctionWithIntersectionTypeParam(t *testing.T) {
	input := `<?php
class Foo {
    public function bar(TranslatorInterface&TranslatorBagInterface&LocaleAwareInterface $translator) {}
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

	if len(nodes) > 0 {
		classNode, ok := nodes[0].(*ast.ClassNode)
		if !ok || len(classNode.Methods) == 0 {
			t.Errorf("Expected ClassNode with methods")
			return
		}
		fn, ok := classNode.Methods[0].(*ast.FunctionNode)
		if !ok || len(fn.Params) == 0 {
			t.Errorf("Expected FunctionNode with parameters")
			return
		}
		param, ok := fn.Params[0].(*ast.ParamNode)
		if !ok {
			t.Errorf("Expected parameter to be *ast.ParamNode, got %T", fn.Params[0])
			return
		}
		if param.IntersectionType == nil {
			t.Errorf("Expected IntersectionTypeNode, got nil")
		} else {
			expected := []string{"TranslatorInterface", "TranslatorBagInterface", "LocaleAwareInterface"}
			for i, typ := range expected {
				if param.IntersectionType.Types[i] != typ {
					t.Errorf("Expected intersection type %d to be '%s', got '%s'", i, typ, param.IntersectionType.Types[i])
				}
			}
		}
	}
}
