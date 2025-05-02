package psr12

import (
	"go-phpcs/sharedcache"
	"go-phpcs/style"
	"go-phpcs/style/generic/functions"
	"strings"
	"sync"
)

// RunAllPSR12Checks runs all PSR-12 style checks on the given file.
// Returns a slice of style.StyleIssue.

// PSR12RuleFunc defines the signature for a PSR-12 rule function
// that returns style issues for a file.
type PSR12RuleFunc func(filename string, content []byte) []style.StyleIssue

// psr12RuleRegistry maps rule codes to their implementation functions
var psr12RuleRegistry = map[string]PSR12RuleFunc{
	"PSR12.Files.EndFileNoTrailingWhitespace": func(filename string, content []byte) []style.StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &NoTrailingWhitespaceChecker{}
		return checker.CheckIssues(lines, filename)
	},
	"PSR12.Files.EndFileNewline": func(filename string, content []byte) []style.StyleIssue {
		if len(content) == 0 || content[len(content)-1] != '\n' {
			return []style.StyleIssue{{
				Filename: filename,
				Line:     0,
				Type:     style.Error,
				Fixable:  true,
				Message:  "File must end with a single blank line (newline)",
				Code:     "PSR12.Files.EndFileNewline",
			}}
		}
		return nil
	},
	"Generic.Functions.DisallowMultipleStatementsSniff": func(filename string, content []byte) []style.StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &functions.DisallowMultipleStatementsSniff{}
		return checker.CheckIssues(lines, filename)
	},
	"PSR12.Files.NoSpaceBeforeSemicolon": func(filename string, content []byte) []style.StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &NoSpaceBeforeSemicolonChecker{}
		return checker.CheckIssues(lines, filename)
	},
	"PSR12.Files.NoBlankLineAfterPHPOpeningTag": func(filename string, content []byte) []style.StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &NoBlankLineAfterPHPOpeningTagChecker{}
		return checker.CheckIssues(lines, filename)
	},
	"PSR12.Classes.OpenBraceOnOwnLine": func(filename string, content []byte) []style.StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &ClassBraceOnOwnLineChecker{}
		return checker.CheckIssues(lines, filename)
	},
	"PSR12.Methods.VisibilityDeclared": func(filename string, content []byte) []style.StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &MethodVisibilityDeclaredChecker{}
		return checker.CheckIssues(lines, filename)
	},
}

// RunSelectedPSR12Checks runs only the selected PSR-12 rules by code. If rules is nil or empty, runs all rules.
func RunSelectedPSR12Checks(filename string, content []byte, rules []string) []style.StyleIssue {
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
			issueCh <- fn(filename, content)
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
	content, err := sharedcache.GetCachedFileContent(filename)
	if err != nil {
		return []style.StyleIssue{{
			Filename: filename,
			Line:     0,
			Type:     style.Error,
			Message:  "[PSR12] Could not load file content: " + err.Error(),
			Code:     "PSR12.Files.FileOpenError",
		}}
	}
	return RunSelectedPSR12Checks(filename, content, nil)
}
