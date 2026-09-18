package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckSymbolIssuesFromCST is the CST-direct analogue of checkSymbolOnNode
// (phpstan_level0_symbols.go): unresolved/abstract/interface instantiation,
// unresolved static/instance method calls (+ visibility/arg-count checks),
// unresolved function calls, unresolved class-const/static-property fetches
// (+ visibility checks), and unresolved property fetches, including the
// "$this inside a static method" checks.
//
// Unlike checkTypeReferenceOnNode/checkLanguageOnNode/
// appendPropertyCallableTypeIssue (which only fire on a handful of discrete
// declaration-shaped constructs), checkSymbolOnNode fires on ordinary
// expression nodes (*ast.NewNode/*ast.FunctionCallNode/*ast.MethodCallNode/
// *ast.ClassConstFetchNode/*ast.PropertyFetchNode) wherever they occur in
// the program, so this port relies on walkSyntaxConfigured's full pre-order
// descent (which already visits every CST node, not just declaration
// slots) rather than dispatching on a small fixed set of kinds.
//
// The one CST-specific wrinkle: a method-call target ($obj->method()) or a
// classic static-call target (Foo::bar()) is represented in the CST as a
// KindMemberAccessExpr/KindStaticMemberAccessExpr node that is a direct
// child of the enclosing KindCallExpr, but ast-side those constructs lower
// straight to *ast.MethodCallNode/*ast.FunctionCallNode (with an
// IdentifierNode Name) - there is no separate PropertyFetchNode/
// ClassConstFetchNode instance for the classic shapes, so lowering that
// child node independently would over-detect. callCallees tracks which
// child nodes were consumed this way so they're skipped when the walker
// later visits them (recursion into their own children, e.g. the receiver
// of a chained ->method(), still happens normally and is unaffected).
// callCallees is keyed by (Green, Offset) rather than the *syntax.RedNode
// pointer itself, since RedNode.Children() allocates a fresh wrapper struct
// on every call - the same logical child visited via two separate
// Children() calls (once here, once inside walkSyntaxConfigured's own
// traversal) would otherwise never compare equal.
// The one exception is a *dynamic* static-call target (Foo::{$m}()/
// Foo::$m()): lowerCallExpr represents that as
// FunctionCallNode{Name: *ast.ClassConstFetchNode{Const: "$", ...}}, and
// walkAllConfigured's *ast.FunctionCallNode case does walk(n.Name, ...),
// so ast-side that ClassConstFetchNode instance genuinely is visited on
// its own (checkSymbolOnNode's *ast.ClassConstFetchNode case then fires
// with the "$"-only Const producing an empty propertyName - an existing
// production quirk, not something to fix here). See
// /memories/repo/cst-direct-migration.md for the full writeup.
//
// A second wrinkle: lowerCallExpr (syntax/lower_expr.go) silently returns
// nil - dropping the call AND its entire argument subtree, however deeply
// nested - whenever its callee can't be represented ast-side: a dynamic
// method-call target ($obj->{$expr}()), a dynamic static-call target with
// no resolvable class part, or (rarer) a callee it can't classify at all.
// This is a genuine, pre-existing lowering-pipeline gap (not a dispatcher
// gap like the switch/yield ones), so e.g. `$this->{$fn}($x::get()->y())`
// makes the entire nested `$x::get()->y()` argument invisible ast-side too
// - checkSymbolOnNode never runs on it, anywhere. suppressed (keyed the
// same (Green, Offset) way as callCallees) marks every descendant of a
// KindCallExpr that fails to lower this way, so this CST-direct port
// mirrors that blindness instead of over-detecting into the dropped
// subtree. See /memories/repo/cst-direct-migration.md for the full writeup.
func CheckSymbolIssuesFromCST(filename string, content []byte, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	return checkSymbolIssuesFromParsed(filename, syntax.Parse(content), ctx, guards)
}

func checkSymbolIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
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

	var issues []AnalysisIssue
	callCallees := map[redNodeKey]bool{}
	suppressed := map[redNodeKey]bool{}
	classCache := map[*syntax.RedNode]*ast.ClassNode{}
	fnCache := map[*syntax.RedNode]*ast.FunctionNode{}
	classCtx := func(n *syntax.RedNode) *ast.ClassNode {
		if n == nil {
			return nil
		}
		if c, ok := classCache[n]; ok {
			return c
		}
		c := syntax.LowerClassLikeContextNode(n, res.File)
		classCache[n] = c
		return c
	}
	fnCtx := func(n *syntax.RedNode) *ast.FunctionNode {
		if n == nil {
			return nil
		}
		if f, ok := fnCache[n]; ok {
			return f
		}
		f := syntax.LowerFunctionLikeContextNode(n, res.File)
		fnCache[n] = f
		return f
	}
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if suppressed[keyOf(n)] {
			return
		}
		switch n.Kind() {
		case syntax.KindCallExpr:
			for _, c := range n.Children() {
				switch c.Kind() {
				case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr, syntax.KindStaticMemberAccessExpr:
					callCallees[keyOf(c)] = true
				}
			}
			node := syntax.LowerExprNode(n, res.File)
			if node == nil {
				syntax.Walk(n, func(d *syntax.RedNode) bool {
					suppressed[keyOf(d)] = true
					return true
				})
				return
			}
			checkSymbolOnNode(filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, guards, &issues)
			if fc, ok := node.(*ast.FunctionCallNode); ok {
				if cf, ok := fc.Name.(*ast.ClassConstFetchNode); ok {
					checkSymbolOnNode(filename, cf, classCtx(class), fnCtx(currentFn), ft, ctx, guards, &issues)
				}
			}
		case syntax.KindNewExpr:
			if node := syntax.LowerExprNode(n, res.File); node != nil {
				checkSymbolOnNode(filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, guards, &issues)
			}
		case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr, syntax.KindStaticMemberAccessExpr:
			if callCallees[keyOf(n)] {
				return
			}
			if node := syntax.LowerExprNode(n, res.File); node != nil {
				checkSymbolOnNode(filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, guards, &issues)
			}
		}
	})
	return issues
}
