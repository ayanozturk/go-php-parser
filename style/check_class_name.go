package style

import (
	"fmt"
	"go-phpcs/ast"
)

// StyleChecker interface is defined in style_checker.go
// ClassNameChecker implements StyleChecker

type ClassNameChecker struct{}

// CheckIssues returns a list of style issues for class names in the given nodes.
func (c *ClassNameChecker) CheckIssues(nodes []ast.Node, filename string) []StyleIssue {
	var issues []StyleIssue
	for _, node := range nodes {
		if cls, ok := node.(*ast.ClassNode); ok {
			if cls.Name != pascalCase(cls.Name) {
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
			if cls.Name != pascalCase(cls.Name) {
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

// pascalCase returns the PascalCase version of the input name.
// If the name is already PascalCase, it returns it unchanged.
func pascalCase(name string) string {
	if name == "" {
		return name
	}
	result := ""
	capitalizeNext := true
	for _, r := range name {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result += string(toUpper(r))
			capitalizeNext = false
		} else {
			result += string(r)
		}
	}
	return result
}


// camelCase returns the camelCase version of the input name.
// If the name is already camelCase, it returns it unchanged.
func camelCase(name string) string {
	if name == "" {
		return name
	}
	// Remove underscores and capitalize following letter
	result := ""
	capitalizeNext := false
	for i, r := range name {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if i == 0 {
			result += string(toLower(r))
			continue
		}
		if capitalizeNext {
			result += string(toUpper(r))
			capitalizeNext = false
		} else {
			result += string(r)
		}
	}
	return result
}

func toLower(r rune) rune {
	if 'A' <= r && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

func toUpper(r rune) rune {
	if 'a' <= r && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}
