package psr12

import (
	"go-phpcs/style"
	"go-phpcs/style/helper"
)

// MethodVisibilityDeclaredChecker checks that all class methods declare visibility (PSR-12 4.2)
type MethodVisibilityDeclaredChecker struct{}

func (c *MethodVisibilityDeclaredChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
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
				issues = append(issues, style.StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     style.Error,
					Fixable:  false,
					Message:  "Visibility must be declared on all class methods",
					Code:     "PSR12.Methods.VisibilityDeclared",
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
