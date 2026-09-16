package style

import (
	"github.com/ayanozturk/go-php-parser/ast"
)

// NoTrailingWhitespaceChecker checks for trailing whitespace at the end of lines (PSR-12 2.2)
type NoTrailingWhitespaceChecker struct{}

func (c *NoTrailingWhitespaceChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	for i, line := range lines {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			runStart := len(line)
			for runStart > 0 {
				ch := line[runStart-1]
				if ch != ' ' && ch != '\t' {
					break
				}
				runStart--
			}
			issues = append(issues, StyleIssue{
				Filename:  filename,
				Line:      i + 1,
				Column:    runStart + 1,
				EndLine:   i + 1,
				EndColumn: len(line) + 1,
				Type:      Error,
				Fixable:   true,
				Message:   "Trailing whitespace detected",
				Code:      "PSR12.Files.EndFileNoTrailingWhitespace",
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
