package analyse

import "github.com/ayanozturk/go-php-parser/ast"

func methodVisibilityIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return ensureStructuralIssues(filename, nodes, ctx).methodVisibilityIssues
}

func throwTypeIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return ensureStructuralIssues(filename, nodes, ctx).throwTypeIssues
}

func ensureStructuralIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	ctx = ensureLevel0Context(filename, nodes, ctx)
	if ctx.hasStructuralIssues {
		return ctx
	}
	fileCtx := analysisFileTypeContext(ctx, nodes)
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendMethodVisibilityOnNode(filename, node, class, currentFn, ft, ctx, &ctx.methodVisibilityIssues)
		appendThrowTypeOnNode(filename, node, ft, ctx, &ctx.throwTypeIssues)
	})
	ctx.hasStructuralIssues = true
	return ctx
}
