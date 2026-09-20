package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckClassModelIssuesFromCST is the CST-direct analogue of
// checkClassModel/appendClassModelOnNode (phpstan_level0_class_model.go):
// final+abstract conflicts, extends/implements legality (unknown parent,
// wrong kind, final/readonly mismatches), abstract/final/private method
// legality, final constant overrides, consistent-constructor legality,
// interface member visibility, readonly property overrides, enum legality,
// and trait-use resolution.
//
// appendClassModelOnNode only fires on 4 concrete declaration-shaped
// ast.Node types (*ast.ClassNode/*ast.InterfaceNode/*ast.TraitUseNode/
// *ast.EnumNode), never on ordinary expressions, and every check it runs
// operates on the FULLY LOWERED node (class.Methods/.Properties/.Constants
// etc. are already-lowered ast.Node slices) — so unlike
// checkSymbolOnNode/checkTypeReferenceOnNode this port never needs to
// re-derive anything from raw CST tokens: it just needs to find the right
// CST node once per declaration and lower that one subtree in full via the
// existing Lower*DeclNode wrappers, then hand the result to the existing,
// unmodified appendClassModelOnNode.
//
// *ast.TraitUseNode (a `use TraitA;` clause inside a class/trait body) is
// not a top-level declaration kind in the CST - lowerClassMembers prepends
// it into ClassNode.Properties (see lower_decl.go), and lowerTraitMembers
// prepends it into TraitNode.Body (see lower_trait.go) - mirroring
// walkAllConfigured's *ast.ClassNode case (which walks n.Properties, so a
// TraitUseNode mixed into that slice reaches appendClassModelOnNode the
// same way a real *ast.PropertyNode would) and its *ast.TraitNode case
// (which walks n.Body the same way). So this port doesn't dispatch on
// KindUseTraitClause directly; it extracts *ast.TraitUseNode entries from
// the already-lowered ClassNode.Properties / TraitNode.Body slices instead.
func CheckClassModelIssuesFromCST(filename string, content []byte, ctx *AnalysisContext) []AnalysisIssue {
	return checkClassModelIssuesFromParsed(filename, sharedParseResult(ctx, content), ctx)
}

func checkClassModelIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext) []AnalysisIssue {
	var issues []AnalysisIssue
	appendDuplicateClassIssues(filename, ctx, &issues)
	if ctx == nil || ctx.Resolver == nil || res == nil || res.File == nil || res.File.Root == nil {
		return issues
	}
	rootFt := ensureSyntaxRootFileTypeContext(ctx, res.File.Root)
	walkSyntaxConfigured(res.File.Root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		switch n.Kind() {
		case syntax.KindClassDecl, syntax.KindAnonymousClass:
			cls := memoLowerClassLike(ctx, n, res.File)
			if cls == nil {
				return
			}
			appendClassModelOnNode(filename, cls, ft, ctx, &issues)
			for _, prop := range cls.Properties {
				if tu, ok := prop.(*ast.TraitUseNode); ok {
					appendClassModelOnNode(filename, tu, ft, ctx, &issues)
				}
			}
		case syntax.KindInterfaceDecl:
			if iface := syntax.LowerInterfaceDeclNode(n, res.File); iface != nil {
				appendClassModelOnNode(filename, iface, ft, ctx, &issues)
			}
		case syntax.KindEnumDecl:
			if enm := syntax.LowerEnumDeclNode(n, res.File); enm != nil {
				appendClassModelOnNode(filename, enm, ft, ctx, &issues)
			}
		case syntax.KindTraitDecl:
			tr := syntax.LowerTraitDeclNode(n, res.File)
			if tr == nil {
				return
			}
			for _, member := range tr.Body {
				if tu, ok := member.(*ast.TraitUseNode); ok {
					appendClassModelOnNode(filename, tu, ft, ctx, &issues)
				}
			}
		}
	})
	return issues
}
