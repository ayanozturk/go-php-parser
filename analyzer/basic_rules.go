package analyzer

import (
	"fmt"
	"go-phpcs/ast"
)

// isBuiltinFunction checks if a function name is a PHP builtin (minimal set for PoC)
func isBuiltinFunction(name string) bool {
	builtins := map[string]bool{
		"echo":   true,
		"print":  true,
		"strlen": true,
		"count":  true,
	}
	return builtins[name]
}

// AnalyzeUnknownFunctionCalls traverses the AST and prints unknown function calls
func AnalyzeUnknownFunctionCalls(nodes []ast.Node) {
	declared := map[string]bool{}
	collectDeclaredFunctions(nodes, declared)
	findUnknownFunctionCalls(nodes, declared)
}

func collectDeclaredFunctions(nodes []ast.Node, declared map[string]bool) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FunctionDecl:
			declared[n.Name] = true
		}
		// Recursively check children if composite node
		if children := getChildren(node); len(children) > 0 {
			collectDeclaredFunctions(children, declared)
		}
	}
}

func findUnknownFunctionCalls(nodes []ast.Node, declared map[string]bool) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FunctionCall:
			if !declared[n.Name] && !isBuiltinFunction(n.Name) {
				fmt.Printf("Unknown function: %s at line %d\n", n.Name, getLine(node))
			}
		}
		if children := getChildren(node); len(children) > 0 {
			findUnknownFunctionCalls(children, declared)
		}
	}
}

// getChildren tries to extract child nodes from a node (simplified for PoC)
func getChildren(node ast.Node) []ast.Node {
	type childProvider interface {
		Children() []ast.Node
	}
	if cp, ok := node.(childProvider); ok {
		return cp.Children()
	}
	return nil
}

// getLine extracts line number from node if available
func getLine(node ast.Node) int {
	type posProvider interface {
		GetPos() ast.Position
	}
	if p, ok := node.(posProvider); ok {
		return p.GetPos().Line
	}
	return 0
}
