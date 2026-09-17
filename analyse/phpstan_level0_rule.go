package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
)

const (
	level0SymbolsCode    = "Level0.Symbols"
	level0ClassModelCode = "Level0.ClassModel"
	level0InvocationCode = "Level0.Invocation"
	level1VariablesCode  = "Level1.Variables"
	level0LanguageCode   = "Level0.Language"
)

type Level0Rule struct{}

func (r *Level0Rule) CheckIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureSharedFileDiagnostics(filename, nodes, ctx)
	return ctx.level0Issues
}

func ensureSharedFileDiagnostics(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	ctx = ensureLevel0Context(filename, nodes, ctx)
	if ctx.hasLevel0Issues {
		return ctx
	}
	return ensureSharedFileDiagnosticsFromCST(filename, ctx.Content, nodes, ctx)
}

func ensureLevel0Context(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	if ctx == nil {
		ctx = &AnalysisContext{}
	}
	if ctx.Resolver == nil {
		ctx.Resolver = BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	}
	return ctx
}

func init() {
	RegisterAnalysisRuleWithLevel(level0SymbolsCode, 0, "level0", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return (&Level0Rule{}).CheckIssues(filename, nodes, ctx)
	})
	RegisterAnalysisRuleWithLevel(level0PropertyCallableTypeCode, 0, "level0", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		ctx = ensureLevel0Context(filename, nodes, ctx)
		_ = (&Level0Rule{}).CheckIssues(filename, nodes, ctx)
		return ctx.level0PropertyCallableIssues
	})
	RegisterAnalysisRuleWithLevel(level1VariablesCode, 1, "level1", checkUndefinedVariables)
}
