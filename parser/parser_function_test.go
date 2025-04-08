package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

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
