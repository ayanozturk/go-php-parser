package psr12

import (
	"go-phpcs/style"
	"regexp"
)

// FunctionCallArgumentSpacingChecker checks spacing between function call arguments
// Implements a simplified version of Generic.Functions.FunctionCallArgumentSpacing
// Flags cases like: foo( 1,  2 ), foo(1 ,2), foo(1,2 )
type FunctionCallArgumentSpacingChecker struct{}

var (
	// Matches function calls with arguments, capturing the argument list
	funcCallRegex = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*)\)`)
	// Matches commas with incorrect spacing
	badCommaSpacing = regexp.MustCompile(`,\s{2,}|\s+,|,\S`)
)

func (c *FunctionCallArgumentSpacingChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	for i, line := range lines {
		matches := funcCallRegex.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			argsStart, argsEnd := m[4], m[5]
			args := line[argsStart:argsEnd]
			if badCommaSpacing.MatchString(args) {
				issues = append(issues, style.StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     style.Error,
					Fixable:  true,
					Message:  "Incorrect spacing between function call arguments",
					Code:     "Generic.Functions.FunctionCallArgumentSpacing",
				})
			}
		}
	}
	return issues
}
