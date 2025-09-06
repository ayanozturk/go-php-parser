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

func TestParseFirstClassCallable(t *testing.T) {
	php := `<?php
$fn = strlen(...);
$callable = foo(...);
$result = $fn("hello");
`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(nodes))
	}

	// Check first assignment: $fn = strlen(...)
	assign1, ok := nodes[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("Expected ExpressionStmt, got %T", nodes[0])
	}
	assignNode1, ok := assign1.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("Expected AssignmentNode, got %T", assign1.Expr)
	}
	callable1, ok := assignNode1.Right.(*ast.FirstClassCallableNode)
	if !ok {
		t.Fatalf("Expected FirstClassCallableNode, got %T", assignNode1.Right)
	}
	if callable1.Name.Value != "strlen" {
		t.Errorf("Expected callable name 'strlen', got '%s'", callable1.Name.Value)
	}

	// Check second assignment: $callable = foo(...)
	assign2, ok := nodes[1].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("Expected ExpressionStmt, got %T", nodes[1])
	}
	assignNode2, ok := assign2.Expr.(*ast.AssignmentNode)
	if !ok {
		t.Fatalf("Expected AssignmentNode, got %T", assign2.Expr)
	}
	callable2, ok := assignNode2.Right.(*ast.FirstClassCallableNode)
	if !ok {
		t.Fatalf("Expected FirstClassCallableNode, got %T", assignNode2.Right)
	}
	if callable2.Name.Value != "foo" {
		t.Errorf("Expected callable name 'foo', got '%s'", callable2.Name.Value)
	}
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

func TestParsePHPDocInFunction(t *testing.T) {
	input := `<?php
/**
 * Test function for PHPDoc parsing
 * @param string $message The message to display
 * @param int $count Number of times to display
 * @return void
 */
function displayMessage($message, $count) {
    for ($i = 0; $i < $count; $i++) {
        echo $message . "\n";
    }
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

	funcNode, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("Expected FunctionNode, got %T", nodes[0])
	}

	// Check function PHPDoc
	if funcNode.PHPDoc == nil {
		t.Error("Expected PHPDoc for function, got nil")
	} else {
		if funcNode.PHPDoc.Description != "Test function for PHPDoc parsing" {
			t.Errorf("Expected function description 'Test function for PHPDoc parsing', got %q", funcNode.PHPDoc.Description)
		}
		if funcNode.PHPDoc.ReturnType != "void" {
			t.Errorf("Expected return type 'void', got %q", funcNode.PHPDoc.ReturnType)
		}
		if len(funcNode.PHPDoc.Params) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(funcNode.PHPDoc.Params))
		} else {
			if funcNode.PHPDoc.Params[0].Name != "message" {
				t.Errorf("Expected first param name 'message', got %q", funcNode.PHPDoc.Params[0].Name)
			}
			if funcNode.PHPDoc.Params[0].Type != "string" {
				t.Errorf("Expected first param type 'string', got %q", funcNode.PHPDoc.Params[0].Type)
			}
			if funcNode.PHPDoc.Params[1].Name != "count" {
				t.Errorf("Expected second param name 'count', got %q", funcNode.PHPDoc.Params[1].Name)
			}
			if funcNode.PHPDoc.Params[1].Type != "int" {
				t.Errorf("Expected second param type 'int', got %q", funcNode.PHPDoc.Params[1].Type)
			}
		}
	}
}
