package psr12

import (
	"go-phpcs/style"
)

// NoSpaceBeforeSemicolonChecker checks that there are no spaces before semicolons (PSR-12 2.4)
type NoSpaceBeforeSemicolonChecker struct{}

func (c *NoSpaceBeforeSemicolonChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	for i, line := range lines {
		trimmed := line
		if len(trimmed) == 0 {
			continue // skip empty lines
		}
		if len(trimmed) > 1 && (trimmed[0:2] == "//" || trimmed[0:1] == "#") {
			continue // skip pure comment lines
		}
		for j := 0; j < len(line); j++ {
			if line[j] == ';' && j > 0 && (line[j-1] == ' ' || line[j-1] == '\t') {
				issues = append(issues, style.StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Column:   j,
					Type:     style.Error,
					Fixable:  true,
					Message:  "Space or tab found before semicolon",
					Code:     "PSR12.Files.NoSpaceBeforeSemicolon",
				})
			}
		}
	}
	return issues
}
