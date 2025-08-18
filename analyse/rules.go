package analyse

import (
	"go-phpcs/ast"
	"sort"
)

type AnalysisIssue struct {
	Filename string
	Line     int
	Column   int
	Code     string
	Message  string
}

type AnalysisRuleFunc func(filename string, nodes []ast.Node) []AnalysisIssue

var (
	analysisRuleRegistry = map[string]AnalysisRuleFunc{}
)

func RegisterAnalysisRule(code string, fn AnalysisRuleFunc) {
	analysisRuleRegistry[code] = fn
}

// ListRegisteredAnalysisRuleCodes returns registered analysis rule codes in sorted order.
func ListRegisteredAnalysisRuleCodes() []string {
	codes := make([]string, 0, len(analysisRuleRegistry))
	for c := range analysisRuleRegistry {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

// ClearAnalysisRules removes all registered analysis rules. Useful for test isolation.
func ClearAnalysisRules() {
	for k := range analysisRuleRegistry {
		delete(analysisRuleRegistry, k)
	}
}

func RunAnalysisRules(filename string, nodes []ast.Node) []AnalysisIssue {
	// Ensure deterministic execution order for testability and reproducibility.
	codes := ListRegisteredAnalysisRuleCodes()

	issues := make([]AnalysisIssue, 0, 8)
	for _, code := range codes {
		fn := analysisRuleRegistry[code]
		issues = append(issues, fn(filename, nodes)...)
	}
	return issues
}
