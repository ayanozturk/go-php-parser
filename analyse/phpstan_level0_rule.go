package analyse

import "go-phpcs/ast"

const (
	level0SymbolsCode    = "PHPStan.Level0.Symbols"
	level0ClassModelCode = "PHPStan.Level0.ClassModel"
	level0InvocationCode = "PHPStan.Level0.Invocation"
	level0VariablesCode  = "PHPStan.Level0.Variables"
	level0LanguageCode   = "PHPStan.Level0.Language"
)

type PHPStanLevel0Rule struct{}

func (r *PHPStanLevel0Rule) CheckIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureLevel0Context(filename, nodes, ctx)
	fileCtx := analysisFileTypeContext(ctx, nodes)
	var issues []AnalysisIssue
	issues = append(issues, r.checkClassModel(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkTypeReferences(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkSymbolsAndCalls(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkUndefinedVariables(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkLanguage(filename, nodes, ctx, fileCtx)...)
	return issues
}

func ensureLevel0Context(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	if ctx == nil {
		ctx = &AnalysisContext{}
	}
	if ctx.Project == nil {
		ctx.Project = BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	}
	if ctx.Resolver == nil {
		ctx.Resolver = ctx.Project
	}
	return ctx
}

func init() {
	RegisterAnalysisRuleWithLevel(level0SymbolsCode, 0, "phpstan.level0", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return (&PHPStanLevel0Rule{}).CheckIssues(filename, nodes, ctx)
	})
}
