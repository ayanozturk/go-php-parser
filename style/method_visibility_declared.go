package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"strings"
)

const methodVisibilityDeclaredCode = "PSR12.Methods.VisibilityDeclared"

type MethodVisibilityDeclaredChecker struct{}

func (c *MethodVisibilityDeclaredChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	inClass := false
	braceDepth := 0
	for i, line := range lines {
		trimmed := helper.TrimWhitespace(line)
		if shouldSkipLine(trimmed) {
			continue
		}
		if helper.IsClassDeclaration(trimmed) {
			inClass = true
			continue
		}
		if inClass {
			braceDepth, inClass = updateBraceState(trimmed, braceDepth, inClass)
			if !inClass {
				continue
			}
			if isMethodDeclaration(trimmed) && !hasVisibility(trimmed) {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     Error,
					Fixable:  false,
					Message:  "Visibility must be declared on all class methods",
					Code:     methodVisibilityDeclaredCode,
				})
			}
		}
	}
	return issues
}

func shouldSkipLine(line string) bool {
	return len(line) > 0 && (line[0] == '/' || line[0] == '*')
}

func updateBraceState(line string, braceDepth int, inClass bool) (int, bool) {
	if line == "{" {
		braceDepth++
		return braceDepth, inClass
	}
	if line == "}" {
		braceDepth--
		if braceDepth <= 0 {
			return braceDepth, false
		}
		return braceDepth, inClass
	}
	return braceDepth, inClass
}

func isMethodDeclaration(line string) bool {
	idx := helper.IndexOfWord(line, "function")
	if idx == -1 {
		return false
	}
	n := idx + len("function")
	for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	if n < len(line) && line[n] == '(' {
		return false
	}
	return true
}

func hasVisibility(line string) bool {
	return helper.HasWord(line, "public") || helper.HasWord(line, "protected") || helper.HasWord(line, "private")
}

func init() {
	RegisterRule(methodVisibilityDeclaredCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &MethodVisibilityDeclaredChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
