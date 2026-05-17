package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
)

// StyleChecker interface is defined in style_checker.go
// ClassNameChecker implements StyleChecker

type ClassNameChecker struct{}

// CheckIssues returns a list of style issues for class names in the given nodes.
func (c *ClassNameChecker) CheckIssues(nodes []ast.Node, filename string) []StyleIssue {
	var issues []StyleIssue
	for _, node := range nodes {
		if cls, ok := node.(*ast.ClassNode); ok {
			if cls.Name != helper.PascalCase(cls.Name) {
				issues = append(issues, StyleIssue{
					Filename:    filename,
					Line:        cls.Pos.Line,
					Column:      cls.Pos.Column,
					Type:        Error,
					Fixable:     false,
					Message:     "Class name should be PascalCase",
					Code:        "PSR1.Classes.ClassDeclaration.PascalCase",
					SubjectKind: "class",
					SubjectName: cls.Name,
				})
			}
		}
	}
	return issues
}

// Deprecated: use CheckIssues for structured output.
func (c *ClassNameChecker) Check(nodes []ast.Node, filename string) {
	_ = c.CheckIssues(nodes, filename)
}

func init() {
	RegisterRule("PSR1.Classes.ClassDeclaration.PascalCase", func(filename string, _ []byte, nodes []ast.Node) []StyleIssue {
		checker := &ClassNameChecker{}
		return checker.CheckIssues(nodes, filename)
	})
}
