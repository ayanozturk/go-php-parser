package psr12

import (
	"bufio"
	"os"
	"sync"
)

// RunAllPSR12Checks runs all PSR-12 style checks on the given file.
// Returns a slice of style.StyleIssue.
import "go-phpcs/style"

// PSR12RuleFunc defines the signature for a PSR-12 rule function
// that returns style issues for a file.
type PSR12RuleFunc func(filename string) []style.StyleIssue

// psr12RuleRegistry maps rule codes to their implementation functions
var psr12RuleRegistry = map[string]PSR12RuleFunc{
	"PSR12.Files.EndFileNoTrailingWhitespace": func(filename string) []style.StyleIssue {
		file, err := os.Open(filename)
		if err != nil {
			return []style.StyleIssue{{
				Filename: filename,
				Line:     0,
				Type:     style.Error,
				Message:  "[PSR12] Could not open file: " + err.Error(),
				Code:     "PSR12.Files.FileOpenError",
			}}
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		checker := &NoTrailingWhitespaceChecker{}
		return checker.CheckIssues(lines, filename)
	},
	"PSR12.Files.EndFileNewline": func(filename string) []style.StyleIssue {
		file, err := os.Open(filename)
		if err != nil {
			return []style.StyleIssue{{
				Filename: filename,
				Line:     0,
				Type:     style.Error,
				Message:  "[PSR12] Could not open file: " + err.Error(),
				Code:     "PSR12.Files.FileOpenError",
			}}
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		checker := &EndFileNewlineChecker{}
		return checker.CheckIssues(lines, filename)
	},
	"PSR12.Files.NoMultipleStatementsPerLine": func(filename string) []style.StyleIssue {
		file, err := os.Open(filename)
		if err != nil {
			return []style.StyleIssue{{
				Filename: filename,
				Line:     0,
				Type:     style.Error,
				Message:  "[PSR12] Could not open file: " + err.Error(),
				Code:     "PSR12.Files.FileOpenError",
			}}
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		checker := &NoMultipleStatementsPerLineChecker{}
		return checker.CheckIssues(lines, filename)
	},
}

// RunSelectedPSR12Checks runs only the selected PSR-12 rules by code. If rules is nil or empty, runs all rules.
func RunSelectedPSR12Checks(filename string, rules []string) []style.StyleIssue {
	var wg sync.WaitGroup
	issueCh := make(chan []style.StyleIssue)
	var ruleFns []PSR12RuleFunc

	if len(rules) == 0 {
		for _, ruleFn := range psr12RuleRegistry {
			ruleFns = append(ruleFns, ruleFn)
		}
	} else {
		for _, ruleCode := range rules {
			if ruleFn, ok := psr12RuleRegistry[ruleCode]; ok {
				ruleFns = append(ruleFns, ruleFn)
			}
		}
	}

	for _, ruleFn := range ruleFns {
		wg.Add(1)
		go func(fn PSR12RuleFunc) {
			defer wg.Done()
			issueCh <- fn(filename)
		}(ruleFn)
	}

	// Close the channel when all goroutines are done
	go func() {
		wg.Wait()
		close(issueCh)
	}()

	var issues []style.StyleIssue
	for iss := range issueCh {
		issues = append(issues, iss...)
	}
	return issues
}

// Existing function for backward compatibility
func RunAllPSR12Checks(filename string) []style.StyleIssue {
	return RunSelectedPSR12Checks(filename, nil)
}
