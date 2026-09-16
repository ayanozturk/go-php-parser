package style

import (
	"strings"
	"unicode"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/style/helper"
)

const psr1MethodCamelCaseCode = "PSR1.Methods.CamelCapsMethodName"

// MethodCamelCaseSniff implements PSR1.Methods.CamelCapsMethodName
// Ensures method names follow camelCase naming convention
type MethodCamelCaseSniff struct{}

// CheckIssues validates method names in the AST nodes
func (s *MethodCamelCaseSniff) CheckIssues(nodes []ast.Node, filename string, content []byte) []StyleIssue {
	lines := SplitLinesCached(content)
	var issues []StyleIssue
	for _, node := range nodes {
		s.checkNodeForMethods(node, filename, lines, &issues)
	}
	return issues
}

// checkNodeForMethods recursively checks AST nodes for method declarations
func (s *MethodCamelCaseSniff) checkNodeForMethods(node ast.Node, filename string, lines []string, issues *[]StyleIssue) {
	switch n := node.(type) {
	case *ast.ClassNode:
		for _, methodNode := range n.Methods {
			if fn, ok := methodNode.(*ast.FunctionNode); ok {
				if !s.isValidCamelCase(fn.Name) {
					s.appendMethodNameIssue(issues, filename, fn.Name, fn.GetPos(), lines)
				}
				// Recursively check method body for nested constructs
				for _, bodyNode := range fn.Body {
					s.checkNodeForMethods(bodyNode, filename, lines, issues)
				}
			}
		}
	case *ast.InterfaceNode:
		for _, memberNode := range n.Members {
			if method, ok := memberNode.(*ast.InterfaceMethodNode); ok {
				if !s.isValidCamelCase(method.Name) {
					s.appendMethodNameIssue(issues, filename, method.Name, method.GetPos(), lines)
				}
			}
		}
	case *ast.TraitNode:
		for _, bodyNode := range n.Body {
			if fn, ok := bodyNode.(*ast.FunctionNode); ok {
				if !s.isValidCamelCase(fn.Name) {
					s.appendMethodNameIssue(issues, filename, fn.Name, fn.GetPos(), lines)
				}
				// Recursively check method body for nested constructs
				for _, methodBodyNode := range fn.Body {
					s.checkNodeForMethods(methodBodyNode, filename, lines, issues)
				}
			}
		}
	}
}

func (s *MethodCamelCaseSniff) appendMethodNameIssue(issues *[]StyleIssue, filename, name string, pos ast.Position, lines []string) {
	lineIdx := pos.Line - 1
	col, endCol := pos.Column, 0
	endLine := pos.Line
	if lineIdx >= 0 && lineIdx < len(lines) {
		if start, end, ok := helper.MethodNameSpanAfterFunction(lines[lineIdx], pos.Column, name); ok {
			col, endCol, endLine = start, end, pos.Line
		}
	}
	*issues = append(*issues, StyleIssue{
		Filename:  filename,
		Line:      pos.Line,
		Column:    col,
		EndLine:   endLine,
		EndColumn: endCol,
		Type:      Error,
		Fixable:   false,
		Message:   "Method name should be camelCase",
		Code:      psr1MethodCamelCaseCode,
	})
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

	first := rune(name[0])
	if !unicode.IsLetter(first) {
		return false
	}

	for _, r := range name[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	return true
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
	RegisterRule(psr1MethodCamelCaseCode, func(filename string, content []byte, nodes []ast.Node) []StyleIssue {
		sniff := &MethodCamelCaseSniff{}
		return sniff.CheckIssues(nodes, filename, content)
	})
}
