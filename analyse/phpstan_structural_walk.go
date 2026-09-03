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
	if ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}
	// Level 2 introduces void.pure. Collect all return-family diagnostics in
	// this existing structural walk from that level onward; individual rule
	// callbacks filter the shared cache by code.
	collectReturn := analysisLevelAtLeast(ctx, 2)
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendMethodVisibilityOnNode(filename, node, class, currentFn, ft, ctx, &ctx.methodVisibilityIssues)
		appendThrowTypeOnNode(filename, node, ft, ctx, &ctx.throwTypeIssues)
		appendPHPDocIssuesOnNode(filename, node, class, ft, ctx, &ctx.phpDocIssues)
		if collectMissingTypes {
			appendMissingTypeIssuesOnNode(filename, node, class, ft, ctx, &ctx.missingTypeIssues)
		}
		if collectReturn {
			appendReturnTypeOnNode(filename, node, class, ft, ctx, &ctx.returnTypeIssues)
		}
	})
	ctx.hasStructuralIssues = true
	if collectReturn {
		ctx.hasReturnTypeIssues = true
	}
	return ctx
}
