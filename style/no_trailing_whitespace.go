package style

import (
	"go-phpcs/ast"
)

// NoTrailingWhitespaceChecker checks for trailing whitespace at the end of lines (PSR-12 2.2)
type NoTrailingWhitespaceChecker struct{}

func (c *NoTrailingWhitespaceChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	for i, line := range lines {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			issues = append(issues, StyleIssue{
				Filename: filename,
				Line:     i + 1,
				Column:   len(line),
				Type:     Error,
				Fixable:  true,
				Message:  "Trailing whitespace detected",
				Code:     "PSR12.Files.EndFileNoTrailingWhitespace",
			})
		}
	}
	return issues
}

func init() {
	RegisterRule("PSR12.Files.EndFileNoTrailingWhitespace", func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := SplitLinesCached(content)
		checker := &NoTrailingWhitespaceChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
