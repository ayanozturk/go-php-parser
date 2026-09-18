package analyse

import (
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckThrowTypeIssuesFromCST is the CST-direct analogue of
// appendThrowTypeOnNode (phpstan_level3_throw.go): validates that a `throw`
// only throws a Throwable-implementing class (and rejects throwing an enum
// or trait name).
//
// appendThrowTypeOnNode ignores class/currentFn scope entirely (it only
// needs FileTypeContext + ctx.Resolver), and only matches *ast.ThrowNode -
// which both the classic `throw $x;` statement (KindThrowStmt, lowered via
// LowerStmtNode) and the PHP 8 `throw` expression form (KindThrowExpr, e.g.
// `$x ?? throw new Exception()`, lowered via LowerExprNode) produce. This
// still drives its traversal through walkSyntaxConfigured (rather than a
// flat syntax.Walk like checkLanguageOnNode/appendPropertyCallableTypeIssue)
// purely for its namespace-boundary-aware FileTypeContext re-derivation -
// class/currentFn/inStatementBody are all ignored.
func CheckThrowTypeIssuesFromCST(filename string, content []byte, ctx *AnalysisContext) []AnalysisIssue {
	return checkThrowTypeIssuesFromParsed(filename, syntax.Parse(content), ctx)
}

func checkThrowTypeIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := CollectFileTypeContextFromSyntax(res.File.Root)

	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		switch n.Kind() {
		case syntax.KindThrowStmt:
			if node := syntax.LowerStmtNode(n, res.File); node != nil {
				appendThrowTypeOnNode(filename, node, ft, ctx, &issues)
			}
		case syntax.KindThrowExpr:
			if node := syntax.LowerExprNode(n, res.File); node != nil {
				appendThrowTypeOnNode(filename, node, ft, ctx, &issues)
			}
		}
	})
	return issues
}
