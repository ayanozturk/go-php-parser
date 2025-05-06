package analyse

import (
	"go-phpcs/ast"
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

func RunAnalysisRules(filename string, nodes []ast.Node) []AnalysisIssue {
	var issues []AnalysisIssue
	for _, fn := range analysisRuleRegistry {
		issues = append(issues, fn(filename, nodes)...)
	}
	return issues
}
