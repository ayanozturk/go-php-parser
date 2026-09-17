package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// checkEmptyStatementIssuesFromCST is the CST-direct analogue of
// appendEmptyStatementIssue, mirroring EmptyStatementRule.CheckIssuesWithSource's
// raw-content branch.
func checkEmptyStatementIssuesFromCST(filename string, content []byte) []AnalysisIssue {
	res := syntax.Parse(content)
	if res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		if n.Kind() == syntax.KindEmptyStmt {
			issues = append(issues, issueSpanRed(filename, n, emptyStatementCode, "Empty statement detected"))
		}
		return true
	})
	return issues
}

// ensureSharedFileDiagnosticsFromCST is the CST-direct analogue of the
// []ast.Node-driven walk in ensureSharedFileDiagnostics: it populates the
// exact same ctx.*Issues fields, using the CST-direct Check*IssuesFromCST
// functions (each already validated 1:1 against its ast.Node counterpart
// across the composer-src/symfony corpora) instead of a single fused
// walkAllWithFileContext pass. guards/phpDocTypeAliases still need to be
// derived from nodes (they have no CST-only equivalent), so nodes remains a
// required parameter here even though issue collection itself no longer
// walks it.
func ensureSharedFileDiagnosticsFromCST(filename string, content []byte, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	fileCtx := analysisFileTypeContext(ctx, nodes)
	guards := collectReflectionGuards(nodes, ctx, fileCtx)

	collectStructural := analysisLevelAtLeast(ctx, 2)
	collectReturn := collectStructural
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)
	if collectStructural && ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}

	var issues []AnalysisIssue
	issues = append(issues, CheckClassModelIssuesFromCST(filename, content, ctx)...)
	issues = append(issues, CheckTypeReferenceIssuesFromCST(filename, content, ctx, guards)...)
	issues = append(issues, CheckSymbolIssuesFromCST(filename, content, ctx, guards)...)
	issues = append(issues, CheckLanguageIssuesFromCST(filename, content)...)
	ctx.level0PropertyCallableIssues = CheckPropertyCallableTypeIssuesFromCST(filename, content)
	ctx.emptyStatementIssues = checkEmptyStatementIssuesFromCST(filename, content)

	if collectStructural {
		ctx.methodVisibilityIssues = CheckMethodVisibilityIssuesFromCST(filename, content, ctx)
		ctx.throwTypeIssues = CheckThrowTypeIssuesFromCST(filename, content, ctx)
		ctx.phpDocIssues = CheckPHPDocIssuesFromCST(filename, content, ctx, ctx.phpDocTypeAliases)
		if collectMissingTypes {
			ctx.missingTypeIssues = CheckMissingTypeIssuesFromCST(filename, content, ctx)
		}
		if collectReturn {
			ctx.returnTypeIssues = CheckReturnTypeIssuesFromCST(filename, content, ctx)
		}
	}

	ctx.level0Issues = issues
	ctx.hasLevel0Issues = true
	if collectStructural {
		ctx.hasStructuralIssues = true
		if collectReturn {
			ctx.hasReturnTypeIssues = true
		}
	}
	return ctx
}

// ensureStructuralIssuesFromCST is the CST-direct analogue of
// ensureStructuralIssues's fallback walk (used when structural issues
// weren't already collected by ensureSharedFileDiagnostics, e.g. because
// ctx.AnalysisLevel was raised between calls).
func ensureStructuralIssuesFromCST(filename string, content []byte, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	if ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}

	collectReturn := analysisLevelAtLeast(ctx, 2)
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)

	ctx.methodVisibilityIssues = CheckMethodVisibilityIssuesFromCST(filename, content, ctx)
	ctx.throwTypeIssues = CheckThrowTypeIssuesFromCST(filename, content, ctx)
	ctx.phpDocIssues = CheckPHPDocIssuesFromCST(filename, content, ctx, ctx.phpDocTypeAliases)
	if collectMissingTypes {
		ctx.missingTypeIssues = CheckMissingTypeIssuesFromCST(filename, content, ctx)
	}
	if collectReturn {
		ctx.returnTypeIssues = CheckReturnTypeIssuesFromCST(filename, content, ctx)
	}

	ctx.hasStructuralIssues = true
	if collectReturn {
		ctx.hasReturnTypeIssues = true
	}
	return ctx
}
