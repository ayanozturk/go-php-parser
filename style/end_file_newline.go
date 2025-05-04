package style

import (
	"go-phpcs/ast"
)

const endFileNewlineCode = "PSR12.Files.EndFileNewline"

type EndFileNewlineChecker struct{}

func (c *EndFileNewlineChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	if len(lines) == 0 {
		return issues
	}
	lastIdx := len(lines) - 1
	secondLastIdx := len(lines) - 2
	last := lines[lastIdx]
	secondLast := ""
	if secondLastIdx >= 0 {
		secondLast = lines[secondLastIdx]
	}
	if last != "" || secondLast == "" || (secondLastIdx > 0 && lines[secondLastIdx-1] == "") {
		issues = append(issues, StyleIssue{
			Filename: filename,
			Line:     len(lines),
			Type:     Error,
			Fixable:  true,
			Message:  "File must end with a single blank line",
			Code:     endFileNewlineCode,
		})
	}
	return issues
}

func init() {
	RegisterRule(endFileNewlineCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		if len(content) == 0 || content[len(content)-1] != '\n' {
			return []StyleIssue{{
				Filename: filename,
				Line:     0,
				Type:     Error,
				Fixable:  true,
				Message:  "File must end with a single blank line (newline)",
				Code:     endFileNewlineCode,
			}}
		}
		return nil
	})
}
