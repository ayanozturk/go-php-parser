package psr12

import (
	"bufio"
	"os"
)

// RunAllPSR12Checks runs all PSR-12 style checks on the given file.
// Returns a slice of error strings.
func RunAllPSR12Checks(filename string) []string {
	var errors []string
	file, err := os.Open(filename)
	if err != nil {
		errors = append(errors, "[PSR12] Could not open file: "+err.Error())
		return errors
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	checker := &NoTrailingWhitespaceChecker{}
	errors = append(errors, checker.Check(lines, filename)...)
	return errors
}
