package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
	"strings"
)

// isAssignmentOperator returns true if the operator is an assignment
func isAssignmentOperator(op token.TokenType) bool {
	switch op {
	case token.T_ASSIGN, token.T_PLUS_EQUAL, token.T_MINUS_EQUAL, token.T_MUL_EQUAL, token.T_DIV_EQUAL, token.T_MOD_EQUAL, token.T_AND_EQUAL, token.T_OR_EQUAL, token.T_CONCAT_EQUAL, token.T_XOR_EQUAL, token.T_COALESCE_EQUAL, token.T_POW_EQUAL:
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
	case *ast.VariableNode, *ast.PropertyFetchNode, *ast.ArrayAccessNode, *ast.ArrayNode:
		return true
	case *ast.ClassConstFetchNode:
		classMember := node.(*ast.ClassConstFetchNode)
		return strings.HasPrefix(classMember.Const, "$")
	default:
		return false
	}
}

// Precedence table for PHP operators (higher number = higher precedence)
var PhpOperatorPrecedence = map[token.TokenType]int{
	token.T_LOGICAL_OR:  0,
	token.T_LOGICAL_XOR: 1,
	token.T_LOGICAL_AND: 2,

	token.T_ASSIGN:         3,
	token.T_PLUS_EQUAL:     3,
	token.T_MINUS_EQUAL:    3,
	token.T_MUL_EQUAL:      3,
	token.T_DIV_EQUAL:      3,
	token.T_MOD_EQUAL:      3,
	token.T_AND_EQUAL:      3,
	token.T_OR_EQUAL:       3,
	token.T_CONCAT_EQUAL:   3,
	token.T_XOR_EQUAL:      3,
	token.T_COALESCE_EQUAL: 3, // ??= assignment
	token.T_POW_EQUAL:      3,

	token.T_QUESTION:    4, // Ternary operator (just above assignment)
	token.T_BOOLEAN_OR:  5, // ||
	token.T_BOOLEAN_AND: 6, // &&
	token.T_PIPE:        7, // |
	token.T_AMPERSAND:   8, // &
	// token.T_XOR_EQUAL:   5, // ^ (already included as assignment above)
	token.T_IS_EQUAL:            9,
	token.T_IS_NOT_EQUAL:        9,
	token.T_IS_IDENTICAL:        9,
	token.T_IS_NOT_IDENTICAL:    9,
	token.T_IS_SMALLER:          10,
	token.T_IS_GREATER:          10,
	token.T_IS_GREATER_OR_EQUAL: 10,
	token.T_IS_SMALLER_OR_EQUAL: 10,
	token.T_SPACESHIP:           10,
	token.T_INSTANCEOF:          11,

	token.T_COALESCE: 12, // ??
	token.T_SL:       12,
	token.T_SR:       12,
	token.T_PLUS:     13,
	token.T_MINUS:    13,
	token.T_DOT:      13,
	token.T_MULTIPLY: 14,
	token.T_DIVIDE:   14,
	token.T_MODULO:   14,
	token.T_POW:      15,
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
	token.T_OR_EQUAL:       true,
	token.T_CONCAT_EQUAL:   true,
	token.T_XOR_EQUAL:      true,
	token.T_COALESCE_EQUAL: true,
	token.T_POW_EQUAL:      true,
	token.T_COALESCE:       true,
	token.T_POW:            true,
}
