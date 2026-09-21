package analyse

import (
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckTypeReferenceIssuesFromCST is the CST-direct analogue of
// checkTypeReferenceOnNode (phpstan_level0_type_refs.go): unresolved
// used-function/const imports, unresolved parameter/return/property/
// constant type references, unresolved caught class names (and
// caught-non-throwable checks), and unresolved attribute classes.
//
// Unlike checkLanguageOnNode/appendPropertyCallableTypeIssue,
// checkTypeReferenceOnNode genuinely needs FileTypeContext (for
// ft.resolveClassLike) and a real *AnalysisContext (for ctx.Resolver), so
// this is the first Phase 3 port to drive its traversal via
// walkSyntaxConfigured rather than a flat syntax.Walk — and the first to
// depend on walkSyntaxConfigured's namespace-boundary FileTypeContext
// re-derivation (see syntax_walk.go/syntax_file_type_context.go). ctx and
// guards are supplied by the caller (a resolver can't be derived from a
// single file's content alone, and reflectionGuards needs a full-file
// ast.Node scan, matching how collectReflectionGuards computes guards up
// front — see syntax_fused_rule.go, which calls collectReflectionGuards
// once and passes the result in, rather than checkTypeReferenceOnNode
// recomputing guards per node).
func CheckTypeReferenceIssuesFromCST(filename string, content []byte, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	return checkTypeReferenceIssuesFromParsed(filename, sharedParseResult(ctx, content), ctx, guards)
}

func checkTypeReferenceIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext, guards reflectionGuards) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)

	var issues []AnalysisIssue
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		switch n.Kind() {
		case syntax.KindUseDecl:
			for _, u := range syntax.LowerUseDeclNode(n, res.File) {
				checkTypeReferenceOnNode(filename, u, ft, ctx, guards, &issues)
			}
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			if class != nil && class.Kind() == syntax.KindInterfaceDecl {
				if im := syntax.LowerInterfaceMethodDeclNode(n, res.File); im != nil {
					checkTypeReferenceOnNode(filename, im, ft, ctx, guards, &issues)
				}
				return
			}
			if fn := memoLowerFunctionDecl(ctx, n, res.File); fn != nil {
				checkTypeReferenceOnNode(filename, fn, ft, ctx, guards, &issues)
			}
		case syntax.KindClosureExpr:
			// Closures lower to *ast.FunctionNode too (see
			// lowerClosureExpr), unlike arrow functions which lower to the
			// distinct *ast.ArrowFunctionNode - checkTypeReferenceOnNode has
			// no case for that type, so arrow-function param/return types
			// are never checked in production either; deliberately not
			// handled here to match.
			if fn := syntax.LowerExprNode(n, res.File); fn != nil {
				checkTypeReferenceOnNode(filename, fn, ft, ctx, guards, &issues)
			}
		case syntax.KindPropertyDecl:
			for _, p := range syntax.LowerPropertyDeclNode(n, res.File) {
				checkTypeReferenceOnNode(filename, p, ft, ctx, guards, &issues)
			}
		case syntax.KindConstDecl:
			// KindClassConstDecl (class-member constants) is deliberately
			// NOT handled here: walkAllConfigured's *ast.ClassNode case
			// (phpstan_level0_walk.go) only recurses into n.Properties and
			// n.Methods, never n.Constants — class constants are silently
			// invisible to every rule driven by that dispatcher, including
			// checkTypeReferenceOnNode. Only KindConstDecl (a global,
			// top-level `const` declaration, lowered as standalone
			// top-level nodes — see lowerTopLevel) is genuinely visited by
			// both paths. Handling KindClassConstDecl here would make the
			// CST-direct path stricter than production; skip it to stay
			// parity-safe until that dispatcher gap is fixed upstream.
			for _, c := range syntax.LowerClassConstDeclNode(n, res.File) {
				checkTypeReferenceOnNode(filename, c, ft, ctx, guards, &issues)
			}
		case syntax.KindCatchClause:
			appendTypeRefCatchIssuesFromCST(filename, n, ft, ctx, guards, &issues)
		case syntax.KindAttribute:
			// Attributes attached to a top-level/namespace-level
			// declaration (class/interface/trait/enum/function) are
			// lowered by lowerTopLevel's KindAttributeList case as
			// standalone SIBLING ast.Node entries (not a field on the
			// declaration they decorate), so they're genuinely visible on
			// the ast side wherever a plain top-level/namespace-level scan
			// would reach them. But an attribute floating inside a
			// statement body (e.g. `return #[Attr] function () {...};`) is
			// never extracted by lowerStmt/lowerClosureExpr at all - it's
			// invisible on the ast side, so skip it here too to stay
			// parity-safe. See /memories/repo/cst-direct-migration.md.
			if inStatementBody {
				return
			}
			if attr := syntax.LowerAttributeNode(n, res.File); attr != nil {
				checkTypeReferenceOnNode(filename, attr, ft, ctx, guards, &issues)
			}
		}
	})
	return issues
}
