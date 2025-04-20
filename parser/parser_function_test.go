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
		if fn, ok := nodes[0].(*ast.FunctionNode); ok {
			if fn.Name != "edgeCase" {
				t.Errorf("Expected function name 'edgeCase', but got '%s'", fn.Name)
			}

			if len(fn.Params) != 2 {
				t.Errorf("Expected 2 parameters, but got %d", len(fn.Params))
			} else {
				if p, ok := fn.Params[0].(*ast.ParamNode); ok {
					if p.Name != "mixed" {
						t.Errorf("Expected parameter 1 to be 'mixed', but got '%s'", p.Name)
					}
					if p.TypeHint != "mixed|null" {
						t.Errorf("Expected parameter 1 type hint to be 'mixed|null', but got '%s'", p.TypeHint)
					}
				}

				if p, ok := fn.Params[1].(*ast.ParamNode); ok {
					if p.Name != "string" {
						t.Errorf("Expected parameter 2 to be 'string', but got '%s'", p.Name)
					}
					if p.TypeHint != "string" {
						t.Errorf("Expected parameter 2 type hint to be 'string', but got '%s'", p.TypeHint)
					}
				}
			}
		}
	}
}
