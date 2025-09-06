package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"strings"
)

const psr1MethodCamelCaseCode = "PSR1.Methods.CamelCapsMethodName"

// MethodCamelCaseSniff implements PSR1.Methods.CamelCapsMethodName
// Ensures method names follow camelCase naming convention
type MethodCamelCaseSniff struct{}

// CheckIssues validates method names in the AST nodes
func (s *MethodCamelCaseSniff) CheckIssues(nodes []ast.Node, filename string) []StyleIssue {
	var issues []StyleIssue
	for _, node := range nodes {
		s.checkNodeForMethods(node, filename, &issues)
	}
	return issues
}

// checkNodeForMethods recursively checks AST nodes for method declarations
func (s *MethodCamelCaseSniff) checkNodeForMethods(node ast.Node, filename string, issues *[]StyleIssue) {
	switch n := node.(type) {
	case *ast.ClassNode:
		for _, methodNode := range n.Methods {
			if fn, ok := methodNode.(*ast.FunctionNode); ok {
				if !s.isValidCamelCase(fn.Name) {
					*issues = append(*issues, StyleIssue{
						Filename: filename,
						Line:     fn.GetPos().Line,
						Column:   fn.GetPos().Column,
						Type:     Error,
						Fixable:  false,
						Message:  "Method name should be camelCase",
						Code:     psr1MethodCamelCaseCode,
					})
				}
				// Recursively check method body for nested constructs
				for _, bodyNode := range fn.Body {
					s.checkNodeForMethods(bodyNode, filename, issues)
				}
			}
		}
	case *ast.InterfaceNode:
		for _, memberNode := range n.Members {
			if method, ok := memberNode.(*ast.InterfaceMethodNode); ok {
				if !s.isValidCamelCase(method.Name) {
					*issues = append(*issues, StyleIssue{
						Filename: filename,
						Line:     method.GetPos().Line,
						Column:   method.GetPos().Column,
						Type:     Error,
						Fixable:  false,
						Message:  "Method name should be camelCase",
						Code:     psr1MethodCamelCaseCode,
					})
				}
			}
		}
	case *ast.TraitNode:
		// Debug: check if trait case is being hit
		_ = n // debug
		for _, bodyNode := range n.Body {
			if fn, ok := bodyNode.(*ast.FunctionNode); ok {
				if !s.isValidCamelCase(fn.Name) {
					*issues = append(*issues, StyleIssue{
						Filename: filename,
						Line:     fn.GetPos().Line,
						Column:   fn.GetPos().Column,
						Type:     Error,
						Fixable:  false,
						Message:  "Method name should be camelCase",
						Code:     psr1MethodCamelCaseCode,
					})
				}
				// Recursively check method body for nested constructs
				for _, methodBodyNode := range fn.Body {
					s.checkNodeForMethods(methodBodyNode, filename, issues)
				}
			}
		}
	}
}

// isValidCamelCase checks if a method name follows camelCase convention
func (s *MethodCamelCaseSniff) isValidCamelCase(name string) bool {
	if name == "" {
		return false
	}

	// Magic methods are exempt from camelCase rules
	if s.isMagicMethod(name) {
		return true
	}

	// Handle method names starting with underscore
	// These are valid if the part after the underscore follows camelCase
	if strings.HasPrefix(name, "_") && len(name) > 1 {
		return s.isValidCamelCase(name[1:])
	}

	// Convert to what camelCase should look like
	expected := helper.CamelCase(name)

	// If the name is already camelCase, it should match the expected
	return name == expected
}

// isMagicMethod checks if the method name is a PHP magic method
func (s *MethodCamelCaseSniff) isMagicMethod(name string) bool {
	magicMethods := []string{
		"__construct", "__destruct", "__call", "__callStatic",
		"__get", "__set", "__isset", "__unset",
		"__sleep", "__wakeup", "__serialize", "__unserialize",
		"__toString", "__invoke", "__set_state", "__clone",
		"__debugInfo", "__autoload",
	}

	for _, magic := range magicMethods {
		if name == magic {
			return true
		}
	}
	return false
}

func init() {
	RegisterRule(psr1MethodCamelCaseCode, func(filename string, _ []byte, nodes []ast.Node) []StyleIssue {
		sniff := &MethodCamelCaseSniff{}
		return sniff.CheckIssues(nodes, filename)
	})
}
