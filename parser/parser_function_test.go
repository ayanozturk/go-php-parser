package parser

import (
	"fmt"
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
		if fn, ok := nodes[0].(*ast.FunctionNode); ok {
			if fn.Name != "edgeCase" {
				t.Errorf("Expected function name 'edgeCase', but got '%s'", fn.Name)
			}

			if len(fn.Params) != 2 {
				t.Errorf("Expected 2 parameters, but got %d", len(fn.Params))
			} else {
				if p, ok := fn.Params[0].(*ast.ParameterNode); ok {
					if p.Name != "mixed" {
						t.Errorf("Expected parameter 1 to be 'mixed', but got '%s'", p.Name)
					}
					if p.TypeHint != "mixed|null" {
						t.Errorf("Expected parameter 1 type hint to be 'mixed|null', but got '%s'", p.TypeHint)
					}
				}

				if p, ok := fn.Params[1].(*ast.ParameterNode); ok {
					if p.Name != "string" {
						t.Errorf("Expected parameter 2 to be 'string', but got '%s'", p.Name)
					}
					if p.TypeHint != "string" {
						t.Errorf("Expected parameter 2 type hint to be 'string', but got '%s'", p.TypeHint)
					}
				}
			}
		} else {
			t.Error("Expected first node to be a FunctionNode")
		}
	}
}

func TestParseSimpleFunction(t *testing.T) {
	input := `<?php
	function sayHello($name) {
		echo "Hello, $name!";
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
		if fn, ok := nodes[0].(*ast.FunctionNode); ok {
			if fn.Name != "sayHello" {
				t.Errorf("Expected function name 'sayHello', but got '%s'", fn.Name)
			}
		} else {
			t.Error("Expected first node to be a FunctionNode")
		}
	}
}

func TestParseFunctionWithVariadicParameter(t *testing.T) {
	input := `<?php
	function sprintf($format, ...$values) {
		return sprintf($format ?? '', ...$values);
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
		if fn, ok := nodes[0].(*ast.FunctionNode); ok {
			if fn.Name != "sprintf" {
				t.Errorf("Expected function name 'sprintf', but got '%s'", fn.Name)
			}

			if len(fn.Params) != 2 {
				t.Errorf("Expected 2 parameters, but got %d", len(fn.Params))
			} else {
				if p, ok := fn.Params[0].(*ast.ParameterNode); ok {
					if p.Name != "format" {
						t.Errorf("Expected parameter 1 to be 'format', but got '%s'", p.Name)
					}
					if p.IsVariadic {
						t.Errorf("Parameter 1 should not be variadic")
					}
				}

				if p, ok := fn.Params[1].(*ast.ParameterNode); ok {
					if p.Name != "values" {
						t.Errorf("Expected parameter 2 to be 'values', but got '%s'", p.Name)
					}
					if !p.IsVariadic {
						t.Errorf("Parameter 2 should be variadic (...$values)")
					}
				}
			}
		} else {
			t.Error("Expected first node to be a FunctionNode")
		}
	}
}

func TestParseFunctionWithByRefParameter(t *testing.T) {
	input := `<?php
	function dumpChildren(string $parent, Profile $profile, &$data) {}
	`

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
		if fn, ok := nodes[0].(*ast.FunctionNode); ok {
			if fn.Name != "dumpChildren" {
				t.Errorf("Expected function name 'dumpChildren', but got '%s'", fn.Name)
			}

			if len(fn.Params) != 3 {
				t.Errorf("Expected 3 parameters, but got %d", len(fn.Params))
			} else {
				if p, ok := fn.Params[2].(*ast.ParameterNode); ok {
					if p.Name != "data" {
						t.Errorf("Expected parameter 3 to be 'data', but got '%s'", p.Name)
					}
					if !p.IsByRef {
						t.Errorf("Expected parameter 3 to be by-reference (&$data)")
					}
				} else {
					t.Errorf("Parameter 3 is not a ParameterNode")
				}
			}
		} else {
			t.Error("Expected first node to be a FunctionNode")
		}
	}
}

func TestParseFunctionWithBackslashTypeHint(t *testing.T) {
	input := `<?php
function test(\Traversable $iterator) {}
`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, got none")
	}

	fn, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("Expected FunctionNode, got %T", nodes[0])
	}

	if fn.Name != "test" {
		t.Errorf("Expected function name 'test', got '%s'", fn.Name)
	}

	if len(fn.Params) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(fn.Params))
	}

	p0 := fn.Params[0].(*ast.ParameterNode)
	if p0.TypeHint != "\\Traversable" {
		t.Errorf("Expected type hint '\\Traversable', got '%s'", p0.TypeHint)
	}
	if p0.Name != "iterator" {
		t.Errorf("Expected parameter name 'iterator', got '%s'", p0.Name)
	}
}

func TestParseFunctionWithDefaultParameters(t *testing.T) {
	input := `<?php
	function greet($greeting = "Hello", $name = "World") {
		echo "$greeting, $name!";
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
		if fn, ok := nodes[0].(*ast.FunctionNode); ok {
			if fn.Name != "greet" {
				t.Errorf("Expected function name 'greet', but got '%s'", fn.Name)
			}

			if len(fn.Params) != 2 {
				t.Errorf("Expected 2 parameters, but got %d", len(fn.Params))
			} else {
				if p, ok := fn.Params[0].(*ast.ParameterNode); ok {
					if p.Name != "greeting" {
						t.Errorf("Expected parameter 1 to be 'greeting', but got '%s'", p.Name)
						fmt.Println(p.String())
					}
				}

				if p, ok := fn.Params[1].(*ast.ParameterNode); ok {
					if p.Name != "name" {
						t.Errorf("Expected parameter 2 to be 'name', but got '%s'", p.Name)
						fmt.Println(p.String())
					}
				}
			}
		} else {
			t.Error("Expected first node to be a FunctionNode")
		}
	}
}
