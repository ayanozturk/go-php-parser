package psr12

import (
	"bufio"
	"os"
)

// RunAllPSR12Checks runs all PSR-12 style checks on the given file.
// Returns a slice of style.StyleIssue.
import "go-phpcs/style"

func RunAllPSR12Checks(filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	file, err := os.Open(filename)
	if err != nil {
		issues = append(issues, style.StyleIssue{
			Filename: filename,
			Line:     0,
			Type:     style.Error,
			Message:  "[PSR12] Could not open file: " + err.Error(),
			Code:     "PSR12.Files.FileOpenError",
		})
		return issues
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	checker := &NoTrailingWhitespaceChecker{}
	issues = append(issues, checker.CheckIssues(lines, filename)...)
	return issues
}
