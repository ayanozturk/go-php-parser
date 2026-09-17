package analyse

import "github.com/ayanozturk/go-php-parser/ast"

func methodVisibilityIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return ensureStructuralIssues(filename, nodes, ctx).methodVisibilityIssues
}

func throwTypeIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return ensureStructuralIssues(filename, nodes, ctx).throwTypeIssues
}

func ensureStructuralIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	ctx = ensureSharedFileDiagnostics(filename, nodes, ctx)
	if ctx.hasStructuralIssues {
		return ctx
	}
	return ensureStructuralIssuesFromCST(filename, ctx.Content, nodes, ctx)
}
