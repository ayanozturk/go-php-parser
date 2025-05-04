package style

import (
	"go-phpcs/ast"
	"strings"
)

type NoSpaceBeforeSemicolonChecker struct{}

func (c *NoSpaceBeforeSemicolonChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
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
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Column:   j,
					Type:     Error,
					Fixable:  true,
					Message:  "Space or tab found before semicolon",
					Code:     "PSR12.Files.NoSpaceBeforeSemicolon",
				})
			}
		}
	}
	return issues
}

func init() {
	RegisterRule("PSR12.Files.NoSpaceBeforeSemicolon", func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &NoSpaceBeforeSemicolonChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
