package analyse

import (
	"fmt"

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
	fileCtx := analysisFileTypeContext(ctx, nodes)
	guards := collectReflectionGuards(nodes, ctx, fileCtx)

	collectStructural := analysisLevelAtLeast(ctx, 2)
	collectReturn := collectStructural
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)
	if collectStructural && ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}

	var typeIssues, symbolIssues, languageIssues, classModelIssues []AnalysisIssue
	appendDuplicateClassIssues(filename, ctx, &classModelIssues)
	labels := map[string]struct{}{}
	var gotos []*ast.GotoNode
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		appendClassModelOnNode(filename, node, ft, ctx, &classModelIssues)
		checkTypeReferenceOnNode(filename, node, ft, ctx, guards, &typeIssues)
		checkSymbolOnNode(filename, node, class, currentFn, ft, ctx, guards, &symbolIssues)
		checkLanguageOnNode(filename, node, ft, labels, &gotos, &languageIssues)
		appendPropertyCallableTypeIssue(filename, node, &ctx.level0PropertyCallableIssues)
		appendEmptyStatementIssue(filename, node, &ctx.emptyStatementIssues)
		if !collectStructural {
			return
		}
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
	for _, goTo := range gotos {
		if _, ok := labels[goTo.Label]; !ok {
			languageIssues = append(languageIssues, issueSpan(filename, goTo, level0LanguageCode, fmt.Sprintf("Goto to undefined label %s.", goTo.Label)))
		}
	}
	issues := classModelIssues
	issues = append(issues, typeIssues...)
	issues = append(issues, symbolIssues...)
	issues = append(issues, languageIssues...)
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
