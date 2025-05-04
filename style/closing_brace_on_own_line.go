// PSR12.Classes.ClosingBraceOnOwnLine
package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"strings"
)

const closingBraceOnOwnLineCode = "PSR12.Classes.ClosingBraceOnOwnLine"

type ClosingBraceOnOwnLineChecker struct{}

func (c *ClosingBraceOnOwnLineChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	if len(lines) == 1 && helper.TrimWhitespace(lines[0]) == "}" {
		issues = append(issues, StyleIssue{
			Filename: filename,
			Line:     1,
			Type:     Error,
			Fixable:  false,
			Message:  "Syntax error: file contains only a closing brace",
			Code:     "Syntax.Error",
		})
		return issues
	}
	for i, line := range lines {
		trimmed := helper.TrimWhitespace(line)
		if trimmed == "}" {
			// Check if next line exists and is not blank (should be blank or EOF)
			if i+1 < len(lines) && helper.TrimWhitespace(lines[i+1]) != "" {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 2,
					Type:     Error,
					Fixable:  false,
					Message:  "Code must not follow closing brace on the same or next line",
					Code:     closingBraceOnOwnLineCode,
				})
			}
			// Check if there is code or comment after the brace on the same line
		} else if idx := strings.Index(line, "}"); idx != -1 {
			after := strings.TrimSpace(line[idx+1:])
			if after != "" {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     Error,
					Fixable:  false,
					Message:  "Closing brace must be on its own line with nothing after",
					Code:     closingBraceOnOwnLineCode,
				})
			}
		}
	}
	return issues
}

func init() {
	RegisterRule(closingBraceOnOwnLineCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &ClosingBraceOnOwnLineChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
