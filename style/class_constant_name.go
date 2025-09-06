package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"regexp"
	"strings"
)

const psr1ClassConstantNameCode = "PSR1.Classes.ClassConstantName"

// ClassConstantNameSniff implements PSR1.Classes.ClassConstantName
// Ensures class constant names follow UPPER_CASE_WITH_UNDERSCORES convention
type ClassConstantNameSniff struct{}

// CheckIssues validates class constant names in the AST nodes
func (s *ClassConstantNameSniff) CheckIssues(nodes []ast.Node, filename string) []StyleIssue {
	var issues []StyleIssue
	for _, node := range nodes {
		s.checkNodeForConstants(node, filename, &issues)
	}
	return issues
}

// checkNodeForConstants recursively checks AST nodes for class constants
func (s *ClassConstantNameSniff) checkNodeForConstants(node ast.Node, filename string, issues *[]StyleIssue) {
	switch n := node.(type) {
	case *ast.ClassNode:
		// Debug: check how many constants are found
		_ = len(n.Constants) // debug
		for _, constantNode := range n.Constants {
			if constant, ok := constantNode.(*ast.ConstantNode); ok {
				// Debug: check constant name validation
				_ = s.isValidConstantName(constant.Name) // debug
				if !s.isValidConstantName(constant.Name) {
					*issues = append(*issues, StyleIssue{
						Filename: filename,
						Line:     constant.GetPos().Line,
						Column:   constant.GetPos().Column,
						Type:     Error,
						Fixable:  false,
						Message:  "Class constant name should be UPPER_CASE_WITH_UNDERSCORES",
						Code:     psr1ClassConstantNameCode,
					})
				}
			}
		}
		// Also check constants in traits
		for _, bodyNode := range n.Methods {
			if fn, ok := bodyNode.(*ast.FunctionNode); ok {
				for _, methodBodyNode := range fn.Body {
					s.checkNodeForConstants(methodBodyNode, filename, issues)
				}
			}
		}
	case *ast.TraitNode:
		for _, bodyNode := range n.Body {
			if fn, ok := bodyNode.(*ast.FunctionNode); ok {
				for _, methodBodyNode := range fn.Body {
					s.checkNodeForConstants(methodBodyNode, filename, issues)
				}
			} else if constant, ok := bodyNode.(*ast.ConstantNode); ok {
				if !s.isValidConstantName(constant.Name) {
					*issues = append(*issues, StyleIssue{
						Filename: filename,
						Line:     constant.GetPos().Line,
						Column:   constant.GetPos().Column,
						Type:     Error,
						Fixable:  false,
						Message:  "Class constant name should be UPPER_CASE_WITH_UNDERSCORES",
						Code:     psr1ClassConstantNameCode,
					})
				}
			}
		}
	}
}

// isValidConstantName checks if a constant name follows UPPER_CASE_WITH_UNDERSCORES convention
func (s *ClassConstantNameSniff) isValidConstantName(name string) bool {
	if name == "" {
		return false
	}

	// Check if name is already in the correct format
	// Should only contain uppercase letters, numbers, and underscores
	validPattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	if !validPattern.MatchString(name) {
		return false
	}

	// Should not start or end with underscore
	if strings.HasPrefix(name, "_") || strings.HasSuffix(name, "_") {
		return false
	}

	// Should not have consecutive underscores
	if strings.Contains(name, "__") {
		return false
	}

	return true
}

// normalizeConstantName converts a name to UPPER_CASE_WITH_UNDERSCORES format
func (s *ClassConstantNameSniff) normalizeConstantName(name string) string {
	// Use the helper function to convert to UPPER_CASE
	upper := helper.CamelCase(name)
	// But we want UPPER_CASE, so convert camelCase to UPPER_CASE
	result := ""
	for i, r := range upper {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += "_"
		}
		if r >= 'a' && r <= 'z' {
			result += string(r - 'a' + 'A')
		} else {
			result += string(r)
		}
	}
	return result
}

func init() {
	RegisterRule(psr1ClassConstantNameCode, func(filename string, _ []byte, nodes []ast.Node) []StyleIssue {
		sniff := &ClassConstantNameSniff{}
		return sniff.CheckIssues(nodes, filename)
	})
}
