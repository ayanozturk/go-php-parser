package psr12

import (
	"go-phpcs/style"
)

// NoMultipleStatementsPerLineChecker checks that there is at most one statement per line (PSR-12 2.3)
type NoMultipleStatementsPerLineChecker struct{}

func (c *NoMultipleStatementsPerLineChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	for i, line := range lines {
		trimmed := line
		if len(trimmed) == 0 {
			continue // skip empty lines
		}
		// Only skip if the line is a pure comment
		ltrim := len(trimmed)
		if ltrim > 1 && (trimmed[0:2] == "//" || trimmed[0:1] == "#") {
			continue
		}
		count := 0
		for j := 0; j < len(line); j++ {
			if line[j] == ';' {
				count++
			}
		}
		if count > 1 {
			issues = append(issues, style.StyleIssue{
				Filename: filename,
				Line:     i + 1,
				Type:     style.Error,
				Fixable:  false,
				Message:  "Multiple statements detected on the same line",
				Code:     "PSR12.Files.NoMultipleStatementsPerLine",
			})
		}
	}
	return issues
}
