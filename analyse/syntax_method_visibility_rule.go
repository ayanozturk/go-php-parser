package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckMethodVisibilityIssuesFromCST is the CST-direct analogue of
// appendMethodVisibilityOnNode (phpstan_level2_method_visibility.go): flags
// calls to a protected method from a class that isn't a subclass (or trait
// user) of the declaring class.
//
// appendMethodVisibilityOnNode only has cases for *ast.FunctionCallNode
// (which also covers the classic Foo::bar() static-call syntax, since its
// callee is a plain string name, never a ClassConstFetchNode-shaped
// expression — see functionCallName/trimFunctionCallNamePrefix) and
// *ast.MethodCallNode. Unlike checkSymbolOnNode, it has no case for a
// standalone property/const fetch, so only syntax.KindCallExpr needs
// dispatching here — no KindMemberAccessExpr/KindStaticMemberAccessExpr
// case, and no "mark the callee consumed" tracking map is needed.
//
// Mirrors checkSymbolOnNode's dynamic-callee blindness: when
// syntax.LowerExprNode can't represent the callee (returns nil for an
// unresolvable dynamic-callee KindCallExpr, e.g. $obj->{$expr}()), the whole
// call and any nested args are invisible ast-side too, so the CST port
// marks that entire subtree suppressed rather than over-detecting into it.
func CheckMethodVisibilityIssuesFromCST(filename string, content []byte, ctx *AnalysisContext) []AnalysisIssue {
	return checkMethodVisibilityIssuesFromParsed(filename, syntax.Parse(content), ctx)
}

func checkMethodVisibilityIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := CollectFileTypeContextFromSyntax(res.File.Root)

	type redNodeKey struct {
		green  *syntax.GreenNode
		offset int
	}
	keyOf := func(n *syntax.RedNode) redNodeKey {
		return redNodeKey{green: n.Green, offset: n.Offset}
	}

	classCache := map[*syntax.RedNode]*ast.ClassNode{}
	fnCache := map[*syntax.RedNode]*ast.FunctionNode{}
	classCtx := func(n *syntax.RedNode) *ast.ClassNode {
		if n == nil {
			return nil
		}
		if v, ok := classCache[n]; ok {
			return v
		}
		v := syntax.LowerClassLikeContextNode(n, res.File)
		classCache[n] = v
		return v
	}
	fnCtx := func(n *syntax.RedNode) *ast.FunctionNode {
		if n == nil {
			return nil
		}
		if v, ok := fnCache[n]; ok {
			return v
		}
		v := syntax.LowerFunctionLikeContextNode(n, res.File)
		fnCache[n] = v
		return v
	}

	var issues []AnalysisIssue
	suppressed := map[redNodeKey]bool{}
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if suppressed[keyOf(n)] {
			return
		}
		if n.Kind() != syntax.KindCallExpr {
			return
		}
		node := syntax.LowerExprNode(n, res.File)
		if node == nil {
			syntax.Walk(n, func(d *syntax.RedNode) bool {
				suppressed[keyOf(d)] = true
				return true
			})
			return
		}
		appendMethodVisibilityOnNode(filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, &issues)
	})
	return issues
}
