package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"strings"
)

const classBraceOnOwnLineCode = "PSR12.Classes.OpenBraceOnOwnLine"

type ClassBraceOnOwnLineChecker struct{}

func (c *ClassBraceOnOwnLineChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	for i := 0; i < len(lines); i++ {
		trimmed := lines[i]
		if helper.IsClassDeclaration(trimmed) {
			if containsBraceOnSameLine(trimmed) {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     Error,
					Fixable:  false,
					Message:  "Opening brace for class-like declaration must be on its own line",
					Code:     classBraceOnOwnLineCode,
				})
				continue
			}
			if i+1 < len(lines) {
				next := lines[i+1]
				if next != "{" {
					issues = append(issues, StyleIssue{
						Filename: filename,
						Line:     i + 2,
						Type:     Error,
						Fixable:  false,
						Message:  "Opening brace for class-like declaration must be on its own line",
						Code:     classBraceOnOwnLineCode,
					})
				}
			}
		}
	}
	return issues
}

func containsBraceOnSameLine(line string) bool {
	keywords := []string{"class ", "interface ", "trait ", "enum "}
	for _, k := range keywords {
		idx := indexOf(line, k)
		if idx != -1 {
			braceIdx := indexOf(line[idx+len(k):], "{")
			if braceIdx != -1 {
				return true
			}
		}
	}
	return false
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func init() {
	RegisterRule(classBraceOnOwnLineCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &ClassBraceOnOwnLineChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
