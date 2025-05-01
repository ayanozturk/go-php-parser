package psr12

import (
	"fmt"
)

// NoTrailingWhitespaceChecker checks for trailing whitespace at the end of lines (PSR-12 2.2)
type NoTrailingWhitespaceChecker struct{}

// Check scans each line of the file for trailing whitespace.
// It returns a slice of error messages with line numbers.
func (c *NoTrailingWhitespaceChecker) Check(lines []string, filename string) []string {
	var errors []string
	for i, line := range lines {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			errors = append(errors, fmt.Sprintf(
				"[PSR12:NoTrailingWhitespace] File: %s | Line: %d | Error: Trailing whitespace detected",
				filename, i+1,
			))
		}
	}
	return errors
}
