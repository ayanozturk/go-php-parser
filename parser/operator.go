package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// isAssignmentOperator returns true if the operator is an assignment
func isAssignmentOperator(op token.TokenType) bool {
	switch op {
	case token.T_ASSIGN, token.T_PLUS_EQUAL, token.T_MINUS_EQUAL, token.T_MUL_EQUAL, token.T_DIV_EQUAL, token.T_MOD_EQUAL, token.T_AND_EQUAL, token.T_CONCAT_EQUAL, token.T_XOR_EQUAL, token.T_COALESCE_EQUAL:
		return true
	default:
		return false
	}
}

// isValidAssignmentTarget returns true if node is a valid assignment target (VariableNode only for now)
func isValidAssignmentTarget(node ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.(type) {
	case *ast.VariableNode, *ast.PropertyFetchNode, *ast.ArrayAccessNode:
		return true
	default:
		return false
	}
}

// Precedence table for PHP operators (higher number = higher precedence)
var PhpOperatorPrecedence = map[token.TokenType]int{
	token.T_ASSIGN:         0,
	token.T_PLUS_EQUAL:     0,
	token.T_MINUS_EQUAL:    0,
	token.T_MUL_EQUAL:      0,
	token.T_DIV_EQUAL:      0,
	token.T_MOD_EQUAL:      0,
	token.T_AND_EQUAL:      0,
	token.T_CONCAT_EQUAL:   0,
	token.T_XOR_EQUAL:      0,
	token.T_COALESCE_EQUAL: 0, // ??= assignment

	token.T_QUESTION:    1, // Ternary operator (just above assignment)
	token.T_BOOLEAN_OR:  2, // ||
	token.T_BOOLEAN_AND: 3, // &&
	token.T_PIPE:        4, // |
	token.T_AMPERSAND:   5, // &
	// token.T_XOR_EQUAL:   5, // ^ (already included as assignment above)
	token.T_IS_EQUAL:            6,
	token.T_IS_NOT_EQUAL:        6,
	token.T_IS_IDENTICAL:        6,
	token.T_IS_NOT_IDENTICAL:    6,
	token.T_IS_SMALLER:          7,
	token.T_IS_GREATER:          7,
	token.T_IS_GREATER_OR_EQUAL: 7,
	token.T_IS_SMALLER_OR_EQUAL: 7,
	token.T_SPACESHIP:           7,
	token.T_INSTANCEOF:          8,

	token.T_COALESCE: 9, // ??
	token.T_PLUS:     10,
	token.T_MINUS:    10,
	token.T_DOT:      10,
	token.T_MULTIPLY: 11,
	token.T_DIVIDE:   11,
	token.T_MODULO:   11,
}

// Operator associativity (true = right-associative)
var PhpOperatorRightAssoc = map[token.TokenType]bool{
	token.T_ASSIGN:         true,
	token.T_PLUS_EQUAL:     true,
	token.T_MINUS_EQUAL:    true,
	token.T_MUL_EQUAL:      true,
	token.T_DIV_EQUAL:      true,
	token.T_MOD_EQUAL:      true,
	token.T_AND_EQUAL:      true,
	token.T_CONCAT_EQUAL:   true,
	token.T_XOR_EQUAL:      true,
	token.T_COALESCE_EQUAL: true,
	token.T_COALESCE:       true,
}
