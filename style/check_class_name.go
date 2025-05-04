package style

import (
	"fmt"
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
					Filename: filename,
					Line:     cls.Pos.Line,
					Column:   cls.Pos.Column,
					Type:     Error,
					Fixable:  false,
					Message:  "Class name should be PascalCase",
					Code:     "PSR1.Classes.ClassDeclaration.PascalCase",
				})
			}
		}
	}
	return issues
}

// Deprecated: use CheckIssues for structured output.
func (c *ClassNameChecker) Check(nodes []ast.Node, filename string) {
	for _, node := range nodes {
		if cls, ok := node.(*ast.ClassNode); ok {
			if cls.Name != helper.PascalCase(cls.Name) {
				// ANSI color codes for pretty output
				const (
					red    = "\033[31m"
					green  = "\033[32m"
					yellow = "\033[33m"
					blue   = "\033[34m"
					reset  = "\033[0m"
					bold   = "\033[1m"
				)

				fmt.Printf("\n%s%sClass Name Style Error%s\n", bold, red, reset)
				fmt.Printf("  %sFile   :%s %s\n", blue, reset, filename)
				fmt.Printf("  %sClass  :%s %s\n", blue, reset, cls.Name)
				fmt.Printf("  %sLine   :%s %d\n", blue, reset, cls.Pos.Line)
				fmt.Printf("  %sColumn :%s %d\n", blue, reset, cls.Pos.Column)
				fmt.Printf("  %sReason :%s Class name should be PascalCase\n\n", yellow, reset)
			}
		}
	}
}

func init() {
	RegisterRule("PSR1.Classes.ClassDeclaration.PascalCase", func(filename string, _ []byte, nodes []ast.Node) []StyleIssue {
		checker := &ClassNameChecker{}
		return checker.CheckIssues(nodes, filename)
	})
}
