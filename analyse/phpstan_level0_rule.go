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
	ctx = ensureLevel0Context(filename, nodes, ctx)
	fileCtx := analysisFileTypeContext(ctx, nodes)
	guards := collectReflectionGuards(nodes, ctx, fileCtx)
	issues := r.checkClassModel(filename, nodes, ctx, fileCtx)

	var typeIssues, symbolIssues, languageIssues []AnalysisIssue
	labels := map[string]struct{}{}
	var gotos []*ast.GotoNode
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		checkTypeReferenceOnNode(filename, node, ft, ctx, guards, &typeIssues)
		checkSymbolOnNode(filename, node, class, currentFn, ft, ctx, guards, &symbolIssues)
		checkLanguageOnNode(filename, node, ft, labels, &gotos, &languageIssues)
	})
	issues = append(issues, typeIssues...)
	issues = append(issues, symbolIssues...)
	for _, goTo := range gotos {
		if _, ok := labels[goTo.Label]; !ok {
			languageIssues = append(languageIssues, issueSpan(filename, goTo, level0LanguageCode, fmt.Sprintf("Goto to undefined label %s.", goTo.Label)))
		}
	}
	issues = append(issues, languageIssues...)
	return issues
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
	RegisterAnalysisRuleWithLevel(level1VariablesCode, 1, "level1", checkUndefinedVariables)
}
