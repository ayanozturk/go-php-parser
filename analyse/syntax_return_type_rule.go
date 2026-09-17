package analyse

import (
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
	res := syntax.Parse(content)
	if res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := CollectFileTypeContextFromSyntax(res.File.Root)

	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		switch n.Kind() {
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			if class != nil && class.Kind() == syntax.KindInterfaceDecl {
				if im := syntax.LowerInterfaceMethodDeclNode(n, res.File); im != nil {
					appendReturnTypeOnNode(filename, im, syntax.LowerClassLikeContextNode(class, res.File), ft, ctx, &issues)
				}
				return
			}
			if fn := syntax.LowerFunctionDeclNode(n, res.File); fn != nil {
				appendReturnTypeOnNode(filename, fn, syntax.LowerClassLikeContextNode(class, res.File), ft, ctx, &issues)
			}
		case syntax.KindClosureExpr:
			if fn := syntax.LowerExprNode(n, res.File); fn != nil {
				appendReturnTypeOnNode(filename, fn, syntax.LowerClassLikeContextNode(class, res.File), ft, ctx, &issues)
			}
		}
	})
	return issues
}
