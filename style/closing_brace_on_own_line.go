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
	inClass := false
	braceDepth := 0
	for i, line := range lines {
		trimmed := helper.TrimWhitespace(line)
		if helper.IsClassDeclaration(trimmed) {
			inClass = true
			braceDepth = 0
			continue
		}
		if inClass {
			// Count braces to find the matching closing brace for the class
			openCount := strings.Count(line, "{")
			closeCount := strings.Count(line, "}")
			braceDepth += openCount
			braceDepth -= closeCount
			if braceDepth < 0 {
				inClass = false
				continue
			}
			if closeCount > 0 && braceDepth == 0 {
				// Find all positions of closing braces
				indices := make([]int, 0)
				for idx := 0; ; {
					pos := strings.Index(line[idx:], "}")
					if pos == -1 {
						break
					}
					indices = append(indices, idx+pos)
					idx += pos + 1
				}
				if len(indices) > 0 {
					lastIdx := indices[len(indices)-1]
					before := strings.TrimSpace(line[:lastIdx])
					after := strings.TrimSpace(line[lastIdx+1:])
					if before != "" || (after != "" && after != "?>") {
						issues = append(issues, StyleIssue{
							Filename: filename,
							Line:     i + 1,
							Type:     Error,
							Fixable:  false,
							Message:  "Class closing brace must be on its own line with nothing before or after",
							Code:     closingBraceOnOwnLineCode,
						})
					}
					// Always check next line for code after class closing brace
					if i+1 < len(lines) {
						nextTrimmed := helper.TrimWhitespace(lines[i+1])
						if nextTrimmed != "" && nextTrimmed != "}" && nextTrimmed != "?>" {
							issues = append(issues, StyleIssue{
								Filename: filename,
								Line:     i + 2,
								Type:     Error,
								Fixable:  false,
								Message:  "Code must not follow class closing brace on the next line (should be blank or another closing brace)",
								Code:     closingBraceOnOwnLineCode,
							})
						}
					}
				}
				inClass = false
				continue
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
