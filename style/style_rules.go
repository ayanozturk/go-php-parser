// Unified style rule registry for all style rules
package style

import (
	"go-phpcs/ast"
	"sort"
	"sync"
)

type RuleFunc func(filename string, content []byte, nodes []ast.Node) []StyleIssue

var (
	ruleRegistryMu sync.RWMutex
	ruleRegistry   = make(map[string]RuleFunc)
)

// RegisterRule registers a style rule by code.
func RegisterRule(code string, fn RuleFunc) {
	ruleRegistryMu.Lock()
	defer ruleRegistryMu.Unlock()
	ruleRegistry[code] = fn
}

// RunSelectedRules runs only the selected rules by code. If rules is nil or empty, runs all rules.
func RunSelectedRules(filename string, content []byte, nodes []ast.Node, rules []string) []StyleIssue {
	ruleRegistryMu.RLock()
	defer ruleRegistryMu.RUnlock()
	var selected []RuleFunc
	if len(rules) == 0 {
		for _, fn := range ruleRegistry {
			selected = append(selected, fn)
		}
	} else {
		for _, code := range rules {
			if fn, ok := ruleRegistry[code]; ok {
				selected = append(selected, fn)
			}
		}
	}
	var issues []StyleIssue
	for _, fn := range selected {
		issues = append(issues, fn(filename, content, nodes)...)
	}
	return issues
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
