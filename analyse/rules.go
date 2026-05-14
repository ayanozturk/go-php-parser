package analyse

import (
	"go-phpcs/ast"
	"sort"
	"sync"
)

type AnalysisIssue struct {
	Filename    string
	Line        int
	Column      int
	Code        string
	Message     string
	SubjectKind string
	SubjectName string
}

type AnalysisRuleFunc func(filename string, nodes []ast.Node) []AnalysisIssue

var (
	analysisRuleRegistry     = map[string]AnalysisRuleFunc{}
	analysisRuleRegistryLock sync.RWMutex
	sortedRuleCodesCache     []string
	sortedRuleCodesDirty     = true
)

func RegisterAnalysisRule(code string, fn AnalysisRuleFunc) {
	analysisRuleRegistryLock.Lock()
	defer analysisRuleRegistryLock.Unlock()

	analysisRuleRegistry[code] = fn
	sortedRuleCodesDirty = true
}

// ListRegisteredAnalysisRuleCodes returns registered analysis rule codes in sorted order.
func ListRegisteredAnalysisRuleCodes() []string {
	analysisRuleRegistryLock.RLock()
	if !sortedRuleCodesDirty {
		codes := append([]string(nil), sortedRuleCodesCache...)
		analysisRuleRegistryLock.RUnlock()
		return codes
	}
	analysisRuleRegistryLock.RUnlock()

	analysisRuleRegistryLock.Lock()
	defer analysisRuleRegistryLock.Unlock()

	if sortedRuleCodesDirty {
		codes := make([]string, 0, len(analysisRuleRegistry))
		for c := range analysisRuleRegistry {
			codes = append(codes, c)
		}
		sort.Strings(codes)
		sortedRuleCodesCache = codes
		sortedRuleCodesDirty = false
	}

	return append([]string(nil), sortedRuleCodesCache...)
}

// ClearAnalysisRules removes all registered analysis rules. Useful for test isolation.
func ClearAnalysisRules() {
	analysisRuleRegistryLock.Lock()
	defer analysisRuleRegistryLock.Unlock()

	for k := range analysisRuleRegistry {
		delete(analysisRuleRegistry, k)
	}
	sortedRuleCodesCache = nil
	sortedRuleCodesDirty = true
}

func RunAnalysisRules(filename string, nodes []ast.Node) []AnalysisIssue {
	codes := ListRegisteredAnalysisRuleCodes()

	issues := make([]AnalysisIssue, 0, 8)
	analysisRuleRegistryLock.RLock()
	defer analysisRuleRegistryLock.RUnlock()
	for _, code := range codes {
		fn := analysisRuleRegistry[code]
		issues = append(issues, fn(filename, nodes)...)
	}
	return issues
}
