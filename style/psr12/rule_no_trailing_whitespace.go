package psr12

import (
	"go-phpcs/style"
)

// NoTrailingWhitespaceChecker checks for trailing whitespace at the end of lines (PSR-12 2.2)
type NoTrailingWhitespaceChecker struct{}

// CheckIssues returns a list of style issues for trailing whitespace.
func (c *NoTrailingWhitespaceChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	for i, line := range lines {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			issues = append(issues, style.StyleIssue{
				Filename: filename,
				Line:     i + 1,
				Column:   len(line),
				Type:     style.Error,
				Fixable:  true,
				Message:  "Trailing whitespace detected",
				Code:     "PSR12.Files.EndFileNoTrailingWhitespace",
			})
		}
	}
	return issues
}

// Deprecated: use CheckIssues for structured output.
func (c *NoTrailingWhitespaceChecker) Check(lines []string, filename string) []string {
	return nil
}
