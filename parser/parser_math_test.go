package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
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
			p := New(l, true)
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

					if binary, ok := assign.Right.(*ast.BinaryExpr); ok {
						// Correctly locate the operator by finding the first non-digit character after the assignment
						operatorIndex := len("<?php $age = ")
						for ; operatorIndex < len(test.input); operatorIndex++ {
							if !('0' <= test.input[operatorIndex] && test.input[operatorIndex] <= '9') && test.input[operatorIndex] != ' ' {
								break
							}
						}
						if binary.Operator != string(test.input[operatorIndex]) {
							t.Errorf("Expected operator '%s', but got '%s'", string(test.input[operatorIndex]), binary.Operator)
						}

						// Adjust left operand validation to match the expected left operand
						if left, ok := binary.Left.(*ast.IntegerNode); ok {
							expectedLeft := int64(test.expectedLeft)
							if left.Value != expectedLeft {
								t.Errorf("Expected left operand to be %d, but got %d", expectedLeft, left.Value)
							}
						}
					}
				} else {
					t.Error("Expected first node to be an AssignmentNode")
				}
			}
		})
	}
}
