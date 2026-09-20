package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckMissingTypeIssuesFromCST is the CST-direct analogue of
// appendMissingTypeIssuesOnNode (phpstan_level6_missing_types.go): flags
// parameters/return types/properties (and their generic/iterable-value
// slots) with no type declared, honoring inherited-contract exemptions.
//
// Fires on the same declaration-shaped kinds as the PHPDoc port
// (KindFunctionDecl/KindMethodDecl/KindClosureExpr/KindPropertyDecl) - the
// KindClosureExpr case is a structural no-op in practice, since
// appendMissingTypeIssuesOnNode's *ast.FunctionNode case requires
// n.Name != "" and a lowered closure's Name is always empty, but it's kept
// for dispatch-shape parity with the PHPDoc/type-refs ports. Unlike the
// PHPDoc port, appendMissingTypeIssuesOnNode never reads
// ctx.phpDocTypeAliases (it derives its own per-declaration alias bindings
// via phpDocTypeAliasBindings(classDoc, doc)), so no extra caller-supplied
// alias parameter is needed here.
func CheckMissingTypeIssuesFromCST(filename string, content []byte, ctx *AnalysisContext) []AnalysisIssue {
	return checkMissingTypeIssuesFromParsed(filename, sharedParseResult(ctx, content), ctx)
}

func checkMissingTypeIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)

	lowerClass := func(class *syntax.RedNode) *ast.ClassNode {
		return memoLowerClassLike(ctx, class, res.File)
	}

	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		switch n.Kind() {
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			if class != nil && class.Kind() == syntax.KindInterfaceDecl {
				if im := syntax.LowerInterfaceMethodDeclNode(n, res.File); im != nil {
					appendMissingTypeIssuesOnNode(filename, im, lowerClass(class), ft, ctx, &issues)
				}
				return
			}
			if fn := memoLowerFunctionDecl(ctx, n, res.File); fn != nil {
				appendMissingTypeIssuesOnNode(filename, fn, lowerClass(class), ft, ctx, &issues)
			}
		case syntax.KindClosureExpr:
			if fn := syntax.LowerExprNode(n, res.File); fn != nil {
				appendMissingTypeIssuesOnNode(filename, fn, lowerClass(class), ft, ctx, &issues)
			}
		case syntax.KindPropertyDecl:
			for _, p := range syntax.LowerPropertyDeclNode(n, res.File) {
				appendMissingTypeIssuesOnNode(filename, p, lowerClass(class), ft, ctx, &issues)
			}
		}
	})
	return issues
}
