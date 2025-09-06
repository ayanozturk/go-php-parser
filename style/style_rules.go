// Unified style rule registry for all style rules
package style

import (
	"go-phpcs/ast"
	"sort"
	"strings"
	"sync"
	"time"
)

type RuleFunc func(filename string, content []byte, nodes []ast.Node) []StyleIssue

var (
	ruleRegistryMu sync.RWMutex
	ruleRegistry   = make(map[string]RuleFunc)
	ruleTimings    = sync.Map{}
)

// RegisterRule registers a style rule by code.
func RegisterRule(code string, fn RuleFunc) {
	ruleRegistryMu.Lock()
	defer ruleRegistryMu.Unlock()
	ruleRegistry[code] = fn
}

// Default rules to run when none are specified - performance-optimized subset
var defaultRules = []string{
	"PSR1.Classes.ClassDeclaration.PascalCase",
	"PSR12.Classes.ClosingBraceOnOwnLine",
	"PSR12.Files.EndFileNewline",
	"PSR12.Files.EndFileNoTrailingWhitespace",
	"PSR12.Files.NoBlankLineAfterPHPOpeningTag",
	"PSR12.Files.NoSpaceBeforeSemicolon",
}

// RunSelectedRules runs only the selected rules by code. If rules is nil or empty, runs default rules.
func RunSelectedRules(filename string, content []byte, nodes []ast.Node, rules []string) []StyleIssue {
	ruleRegistryMu.RLock()
	defer ruleRegistryMu.RUnlock()
	var ruleCodes []string

	if len(rules) == 0 {
		// Use default rules instead of all rules for better performance
		ruleCodes = defaultRules
	} else {
		for _, code := range rules {
			if code == "all" {
				// Special case: run all registered rules
				for code := range ruleRegistry {
					ruleCodes = append(ruleCodes, code)
				}
				break // No need to process other rules
			} else if _, ok := ruleRegistry[code]; ok {
				ruleCodes = append(ruleCodes, code)
			}
		}
	}

	// Categorize rules by performance for optimized execution
	fastRules, slowRules := categorizeRulesByPerformance(ruleCodes)

	var issues []StyleIssue

	// Run all rules sequentially with optimized ordering (fast first, slow last)
	allRules := append(fastRules, slowRules...)
	for _, code := range allRules {
		if fn, ok := ruleRegistry[code]; ok {
			start := time.Now()
			ruleIssues := fn(filename, content, nodes)
			duration := time.Since(start)
			ruleTimings.Store(code, duration)
			issues = append(issues, ruleIssues...)
		}
	}

	return issues
}

// categorizeRulesByPerformance separates rules into fast and slow categories
func categorizeRulesByPerformance(ruleCodes []string) ([]string, []string) {
	var fastRules, slowRules []string

	// Known slow rules that do expensive operations
	slowRulePatterns := []string{
		"PSR1.Files.SideEffects",   // Reads entire file content
		"PSR12.ControlStructures.", // Complex control structure analysis
		"Squiz.ControlStructures.", // Complex control structure analysis
		"Generic.Functions.",       // Function analysis
		"Squiz.Functions.",         // Function analysis
		"PSR1.Methods.",            // Method analysis
		"Squiz.Classes.",           // Class analysis
	}

	for _, code := range ruleCodes {
		isSlow := false
		for _, pattern := range slowRulePatterns {
			if strings.Contains(code, pattern) {
				isSlow = true
				break
			}
		}
		if isSlow {
			slowRules = append(slowRules, code)
		} else {
			fastRules = append(fastRules, code)
		}
	}

	return fastRules, slowRules
}

// runRulesParallel executes rules in parallel for better performance
func runRulesParallel(ruleCodes []string, registry map[string]RuleFunc, filename string, content []byte, nodes []ast.Node) []StyleIssue {
	if len(ruleCodes) == 0 {
		return nil
	}

	type ruleResult struct {
		issues []StyleIssue
	}

	results := make(chan ruleResult, len(ruleCodes))
	var wg sync.WaitGroup

	for _, code := range ruleCodes {
		if fn, ok := registry[code]; ok {
			wg.Add(1)
			go func(fn RuleFunc, code string) {
				defer wg.Done()
				start := time.Now()
				ruleIssues := fn(filename, content, nodes)
				duration := time.Since(start)
				ruleTimings.Store(code, duration)
				results <- ruleResult{issues: ruleIssues}
			}(fn, code)
		}
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	var allIssues []StyleIssue
	for result := range results {
		allIssues = append(allIssues, result.issues...)
	}

	return allIssues
}

// createBatches splits a slice into batches of specified size
func createBatches(rules []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(rules); i += batchSize {
		end := i + batchSize
		if end > len(rules) {
			end = len(rules)
		}
		batches = append(batches, rules[i:end])
	}
	return batches
}

// GetRuleTimings returns performance statistics for rules
func GetRuleTimings() map[string]time.Duration {
	timings := make(map[string]time.Duration)
	ruleTimings.Range(func(key, value interface{}) bool {
		timings[key.(string)] = value.(time.Duration)
		return true
	})
	return timings
}

// ListRegisteredRuleCodes returns a sorted list of all registered rule codes.
func ListRegisteredRuleCodes() []string {
	ruleRegistryMu.RLock()
	defer ruleRegistryMu.RUnlock()
	codes := make([]string, 0, len(ruleRegistry))
	for code := range ruleRegistry {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}
