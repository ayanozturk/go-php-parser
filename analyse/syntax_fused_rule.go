package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
)

// ensureSharedFileDiagnosticsFromCST is the CST-direct analogue of the
// []ast.Node-driven walk in ensureSharedFileDiagnostics: it populates the
// exact same ctx.*Issues fields, using one walkSyntaxConfigured plus one
// flat syntax.Walk (see syntax_fused_walk.go) instead of separate per-rule
// traversals. guards/phpDocTypeAliases still need to be derived from nodes
// (they have no CST-only equivalent), so nodes remains a required parameter
// here even though issue collection itself no longer walks it.
func ensureSharedFileDiagnosticsFromCST(filename string, content []byte, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	res := sharedParseResult(ctx, content)
	fileCtx := analysisFileTypeContext(ctx, nodes)
	guards := collectReflectionGuards(nodes, ctx, fileCtx)

	collectStructural := analysisLevelAtLeast(ctx, 2)
	collectReturn := collectStructural
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)
	if collectStructural && ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}

	var level0 []AnalysisIssue
	appendDuplicateClassIssues(filename, ctx, &level0)

	var methodVisibility, throwType, phpDoc, missingType, returnType []AnalysisIssue
	if res != nil && res.File != nil && res.File.Root != nil {
		rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
		runFusedConfiguredCSTWalk(res.File.Root, rootFt, fusedConfiguredCSTOpts{
			filename:            filename,
			res:                 res,
			ctx:                 ctx,
			guards:              guards,
			level0:              &level0,
			collectLevel0:       true,
			collectStructural:   collectStructural,
			collectMissingTypes: collectMissingTypes,
			collectReturn:       collectReturn,
			methodVisibility:    &methodVisibility,
			throwType:           &throwType,
			phpDoc:              &phpDoc,
			missingType:         &missingType,
			returnType:          &returnType,
		})

		flat := collectFusedFlatCSTDiagnostics(filename, res)
		level0 = append(level0, flat.language...)
		ctx.level0PropertyCallableIssues = flat.propertyCallable
		ctx.emptyStatementIssues = flat.emptyStatement
	}

	ctx.level0Issues = level0
	ctx.hasLevel0Issues = true
	if collectStructural {
		ctx.methodVisibilityIssues = methodVisibility
		ctx.throwTypeIssues = throwType
		ctx.phpDocIssues = phpDoc
		if collectMissingTypes {
			ctx.missingTypeIssues = missingType
		}
		if collectReturn {
			ctx.returnTypeIssues = returnType
		}
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
	res := sharedParseResult(ctx, content)
	if ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}

	collectReturn := analysisLevelAtLeast(ctx, 2)
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)

	var methodVisibility, throwType, phpDoc, missingType, returnType []AnalysisIssue
	if res != nil && res.File != nil && res.File.Root != nil {
		rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
		runFusedConfiguredCSTWalk(res.File.Root, rootFt, fusedConfiguredCSTOpts{
			filename:            filename,
			res:                 res,
			ctx:                 ctx,
			collectStructural:   true,
			collectMissingTypes: collectMissingTypes,
			collectReturn:       collectReturn,
			methodVisibility:    &methodVisibility,
			throwType:           &throwType,
			phpDoc:              &phpDoc,
			missingType:         &missingType,
			returnType:          &returnType,
		})
	}

	ctx.methodVisibilityIssues = methodVisibility
	ctx.throwTypeIssues = throwType
	ctx.phpDocIssues = phpDoc
	if collectMissingTypes {
		ctx.missingTypeIssues = missingType
	}
	if collectReturn {
		ctx.returnTypeIssues = returnType
	}

	ctx.hasStructuralIssues = true
	if collectReturn {
		ctx.hasReturnTypeIssues = true
	}
	return ctx
}
