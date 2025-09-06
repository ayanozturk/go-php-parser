package style

import (
	"go-phpcs/ast"
)

type NoBlankLineAfterPHPOpeningTagChecker struct{}

func (c *NoBlankLineAfterPHPOpeningTagChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	for i, line := range lines {
		if line == "<?php" && i+1 < len(lines) {
			if lines[i+1] != "" {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 2, // the line after <?php
					Type:     Error,
					Fixable:  true,
					Message:  "Missing blank line after opening <?php tag",
					Code:     "PSR12.Files.MissingBlankLineAfterPHPOpeningTag",
				})
			}
			break // Only check the first opening tag
		}
	}
	return issues
}

func init() {
	RegisterRule("PSR12.Files.NoBlankLineAfterPHPOpeningTag", func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := SplitLinesCached(content)
		checker := &NoBlankLineAfterPHPOpeningTagChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
