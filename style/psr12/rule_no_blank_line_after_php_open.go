package psr12

import (
	"go-phpcs/style"
	"strings"
)

// NoBlankLineAfterPHPOpeningTagChecker checks that there is a blank line after the opening <?php tag (PSR-12 2.2)
type NoBlankLineAfterPHPOpeningTagChecker struct{}

func (c *NoBlankLineAfterPHPOpeningTagChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	for i, line := range lines {
		if line == "<?php" && i+1 < len(lines) {
			if lines[i+1] != "" {
				issues = append(issues, style.StyleIssue{
					Filename: filename,
					Line:     i + 2, // the line after <?php
					Type:     style.Error,
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
	RegisterPSR12Rule("PSR12.Files.NoBlankLineAfterPHPOpeningTag", func(filename string, content []byte) []style.StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &NoBlankLineAfterPHPOpeningTagChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
