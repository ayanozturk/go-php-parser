package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckReturnTypeIssuesFromCST is the CST-direct analogue of
// appendReturnTypeOnNode (return_type_rule.go): checks declared vs actual
// return types, void/never-function issues, and return-completeness for
// function/method/closure declarations.
//
// Fires on KindFunctionDecl/KindMethodDecl/KindClosureExpr, mirroring the
// PHPDoc/missing-types ports' dispatch shape. Unlike those two,
// appendReturnTypeOnNode only ever matches *ast.FunctionNode via a type
// assertion and no-ops for anything else - so for an interface method
// (which lowers to *ast.InterfaceMethodNode via LowerInterfaceMethodDeclNode,
// not *ast.FunctionNode), passing the lowered node straight through is
// already a correct, harmless no-op exactly matching ast-side behavior
// (ast-side's walkAllConfigured also visits *ast.InterfaceMethodNode nodes
// and appendReturnTypeOnNode silently ignores them) - no extra skip logic
// needed. No KindPropertyDecl case: return-type checking never applies to
// properties.
func CheckReturnTypeIssuesFromCST(filename string, content []byte, ctx *AnalysisContext) []AnalysisIssue {
	return checkReturnTypeIssuesFromParsed(filename, sharedParseResult(ctx, content), ctx)
}

func checkReturnTypeIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := CollectFileTypeContextFromSyntax(res.File.Root)

	// lowerClassCache memoizes LowerClassLikeContextNode per class within this
	// single walk: the walker passes the same *syntax.RedNode class pointer to
	// every member of that class, so without this cache a class with N members
	// re-lowers its entire body (all N members) on each of the N member visits
	// - O(N^2) work per class instead of O(N).
	lowerClassCache := map[*syntax.RedNode]*ast.ClassNode{}
	lowerClass := func(class *syntax.RedNode) *ast.ClassNode {
		if class == nil {
			return nil
		}
		if cls, ok := lowerClassCache[class]; ok {
			return cls
		}
		cls := syntax.LowerClassLikeContextNode(class, res.File)
		lowerClassCache[class] = cls
		return cls
	}

	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		switch n.Kind() {
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			if class != nil && class.Kind() == syntax.KindInterfaceDecl {
				if im := syntax.LowerInterfaceMethodDeclNode(n, res.File); im != nil {
					appendReturnTypeOnNode(filename, im, lowerClass(class), ft, ctx, &issues)
				}
				return
			}
			if fn := syntax.LowerFunctionDeclNode(n, res.File); fn != nil {
				appendReturnTypeOnNode(filename, fn, lowerClass(class), ft, ctx, &issues)
			}
		case syntax.KindClosureExpr:
			if fn := syntax.LowerExprNode(n, res.File); fn != nil {
				appendReturnTypeOnNode(filename, fn, lowerClass(class), ft, ctx, &issues)
			}
		}
	})
	return issues
}
