package style

import (
	"go-phpcs/ast"
	"regexp"
	"strings"
)

type FunctionCallArgumentSpacingChecker struct{}

var (
	funcCallRegex   = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*)\)`)
	badCommaSpacing = regexp.MustCompile(`,\s{2,}|\s+,|,\S`)
)

func (c *FunctionCallArgumentSpacingChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	for i, line := range lines {
		matches := funcCallRegex.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			argsStart, argsEnd := m[4], m[5]
			args := line[argsStart:argsEnd]
			if badCommaSpacing.MatchString(args) {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     Error,
					Fixable:  true,
					Message:  "Incorrect spacing between function call arguments",
					Code:     "Generic.Functions.FunctionCallArgumentSpacing",
				})
			}
		}
	}
	return issues
}

func init() {
	RegisterRule("Generic.Functions.FunctionCallArgumentSpacing", func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &FunctionCallArgumentSpacingChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
