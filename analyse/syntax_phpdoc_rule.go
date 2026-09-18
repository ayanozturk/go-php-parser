package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckPHPDocIssuesFromCST is the CST-direct analogue of
// appendPHPDocIssuesOnNode (phpstan_level2_phpdoc.go): PHPDoc @param/
// @return/@var type issues (unresolved classes, generic argument-count/
// bound mismatches, incompatibility with the native type hint) for
// functions, methods, interface method signatures, and properties.
//
// Fires on the same 4 declaration-shaped kinds as checkTypeReferenceOnNode
// (KindFunctionDecl/KindMethodDecl distinguished by the enclosing class-
// like's kind, KindClosureExpr, KindPropertyDecl — see that port's doc
// comment for the interface-vs-class-method distinction and the closure
// lowering note); no other CST kind needs dispatching since
// appendPHPDocIssuesOnNode's switch has no other case.
//
// phpDocAliases mirrors how checkTypeReferenceOnNode's guards parameter is
// supplied by the caller: collectPHPDocTypeAliases needs a full-file
// ast.Node scan that can't be derived from a single CST node in isolation.
// It's only assigned into ctx.phpDocTypeAliases when not already cached
// there, mirroring ensureStructuralIssues's own caching (so a shared ctx
// reused across multiple CST-direct rule calls for the same file only pays
// for this once).
func CheckPHPDocIssuesFromCST(filename string, content []byte, ctx *AnalysisContext, phpDocAliases map[string]struct{}) []AnalysisIssue {
	return checkPHPDocIssuesFromParsed(filename, syntax.Parse(content), ctx, phpDocAliases)
}

func checkPHPDocIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext, phpDocAliases map[string]struct{}) []AnalysisIssue {
	if ctx == nil {
		return nil
	}
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	if ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = phpDocAliases
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
					appendPHPDocIssuesOnNode(filename, im, lowerClass(class), ft, ctx, &issues)
				}
				return
			}
			if fn := syntax.LowerFunctionDeclNode(n, res.File); fn != nil {
				appendPHPDocIssuesOnNode(filename, fn, lowerClass(class), ft, ctx, &issues)
			}
		case syntax.KindClosureExpr:
			if fn := syntax.LowerExprNode(n, res.File); fn != nil {
				appendPHPDocIssuesOnNode(filename, fn, lowerClass(class), ft, ctx, &issues)
			}
		case syntax.KindPropertyDecl:
			for _, p := range syntax.LowerPropertyDeclNode(n, res.File) {
				appendPHPDocIssuesOnNode(filename, p, lowerClass(class), ft, ctx, &issues)
			}
		}
	})
	return issues
}
