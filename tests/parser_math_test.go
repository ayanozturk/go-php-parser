package tests

import (
	"testing"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
)

func TestParserMathExpression(t *testing.T) {
	tests := []struct {
		input          string
		expectedLeft   int
		expectedRight  int
		operator       string
		expectedResult int
	}{
		{"<?php $age = 20 + 5;", 20, 5, "+", 25},
		{"<?php $age = 10 + 15;", 10, 15, "+", 25},
		{"<?php $age = 30 - 5;", 30, 5, "-", 25},
		{"<?php $age = 5 * 5;", 5, 5, "*", 25},
		{"<?php $age = 50 / 2;", 50, 2, "/", 25},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			l := lexer.New(test.input)
			p := parser.New(l, true)
			nodes := p.Parse()
			if len(p.Errors()) > 0 {
				t.Errorf("Parser returned errors: %v", p.Errors())
			}
			if len(nodes) == 0 {
				t.Error("Expected at least one node, but got none")
			}
			if len(nodes) > 0 {
				if assign, ok := nodes[0].(*ast.AssignmentNode); ok {
					if variable, ok := assign.Left.(*ast.VariableNode); ok {
						if variable.Name != "age" {
							t.Errorf("Expected variable name 'age', but got '%s'", variable.Name)
						}
					} else {
						t.Error("Expected left side of assignment to be a VariableNode")
					}

					if _, ok := assign.Right.(*ast.BinaryExpr); ok {
						operatorIndex := len("<?php $age = ")
						for ; operatorIndex < len(test.input); operatorIndex++ {
							if !('0' <= test.input[operatorIndex] && test.input[operatorIndex] <= '9') && test.input[operatorIndex] != ' ' {
								break
							}
						}
						operator := string(test.input[operatorIndex])
						if operator != test.operator {
							t.Errorf("Expected operator '%s', got '%s'", test.operator, operator)
						}
					} else {
						t.Error("Expected right side of assignment to be a BinaryExpr")
					}
				}
			}
		})
	}
}
