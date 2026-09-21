package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

type cstRedNodeKey struct {
	green  *syntax.GreenNode
	offset int
}

func cstKeyOf(n *syntax.RedNode) cstRedNodeKey {
	return cstRedNodeKey{green: n.Green, offset: n.Offset}
}

type fusedFlatCSTOut struct {
	language         []AnalysisIssue
	propertyCallable []AnalysisIssue
	emptyStatement   []AnalysisIssue
}

func collectFusedFlatCSTDiagnostics(filename string, res *syntax.ParseResult) fusedFlatCSTOut {
	out := fusedFlatCSTOut{}
	if res == nil || res.File == nil || res.File.Root == nil {
		return out
	}

	labels := map[string]struct{}{}
	var gotos []languageCSTGoto
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindEmptyStmt:
			out.emptyStatement = append(out.emptyStatement, issueSpanRed(filename, n, emptyStatementCode, "Empty statement detected"))
		case syntax.KindSwitchStmt:
			return false
		case syntax.KindLabelStmt, syntax.KindGotoStmt,
			syntax.KindArrayExpr, syntax.KindUnaryExpr, syntax.KindIncludeExpr, syntax.KindCastExpr:
			appendLanguageNonCallIssuesFromCST(filename, n, labels, &gotos, &out.language)
		case syntax.KindPropertyDecl:
			appendPropertyCallableTypeIssueFromCST(filename, n, &out.propertyCallable)
		case syntax.KindParam:
			appendPromotedParamCallableTypeIssueFromCST(filename, n, &out.propertyCallable)
		case syntax.KindCallExpr:
			appendLanguageCallIssuesFromCST(filename, n, &out.language)
		}
		return true
	})
	appendUndefinedGotoIssuesFromCST(filename, labels, gotos, &out.language)
	return out
}

type fusedConfiguredCSTOpts struct {
	filename            string
	res                 *syntax.ParseResult
	ctx                 *AnalysisContext
	guards              reflectionGuards
	level0              *[]AnalysisIssue
	collectLevel0       bool
	collectStructural   bool
	collectMissingTypes bool
	collectReturn       bool
	methodVisibility    *[]AnalysisIssue
	throwType           *[]AnalysisIssue
	phpDoc              *[]AnalysisIssue
	missingType         *[]AnalysisIssue
	returnType          *[]AnalysisIssue
}

func runFusedConfiguredCSTWalk(root *syntax.RedNode, rootFt FileTypeContext, o fusedConfiguredCSTOpts) {
	if root == nil {
		return
	}
	res := o.res
	ctx := o.ctx

	callCallees := map[cstRedNodeKey]bool{}
	symbolSuppressed := map[cstRedNodeKey]bool{}
	visibilitySuppressed := map[cstRedNodeKey]bool{}

	classCtx := func(n *syntax.RedNode) *ast.ClassNode {
		return memoLowerClassLike(ctx, n, res.File)
	}
	fnCtx := func(n *syntax.RedNode) *ast.FunctionNode {
		return memoLowerFunctionLike(ctx, n, res.File)
	}
	lowerClass := func(class *syntax.RedNode) *ast.ClassNode {
		return memoLowerClassLike(ctx, class, res.File)
	}

	runClassModel := o.collectLevel0 && ctx != nil && ctx.Resolver != nil
	runTypeRefs := o.collectLevel0
	runSymbols := o.collectLevel0

	walkSyntaxConfigured(root, rootFt, func(n, class, currentFn *syntax.RedNode, ft FileTypeContext, inStatementBody bool) {
		if runClassModel {
			switch n.Kind() {
			case syntax.KindClassDecl, syntax.KindAnonymousClass:
				cls := memoLowerClassLike(ctx, n, res.File)
				if cls == nil {
					break
				}
				appendClassModelOnNode(o.filename, cls, ft, ctx, o.level0)
				for _, prop := range cls.Properties {
					if tu, ok := prop.(*ast.TraitUseNode); ok {
						appendClassModelOnNode(o.filename, tu, ft, ctx, o.level0)
					}
				}
			case syntax.KindInterfaceDecl:
				if iface := syntax.LowerInterfaceDeclNode(n, res.File); iface != nil {
					appendClassModelOnNode(o.filename, iface, ft, ctx, o.level0)
				}
			case syntax.KindEnumDecl:
				if enm := syntax.LowerEnumDeclNode(n, res.File); enm != nil {
					appendClassModelOnNode(o.filename, enm, ft, ctx, o.level0)
				}
			case syntax.KindTraitDecl:
				tr := syntax.LowerTraitDeclNode(n, res.File)
				if tr != nil {
					for _, member := range tr.Body {
						if tu, ok := member.(*ast.TraitUseNode); ok {
							appendClassModelOnNode(o.filename, tu, ft, ctx, o.level0)
						}
					}
				}
			}
		}

		if runTypeRefs {
			switch n.Kind() {
			case syntax.KindUseDecl:
				appendTypeRefUseIssuesFromCST(o.filename, n, ctx, o.guards, o.level0)
			case syntax.KindFunctionDecl, syntax.KindMethodDecl:
				if class != nil && class.Kind() == syntax.KindInterfaceDecl {
					if im := memoLowerInterfaceMethod(ctx, n, res.File); im != nil {
						checkTypeReferenceOnNode(o.filename, im, ft, ctx, o.guards, o.level0)
					}
				} else if fn := memoLowerFunctionDecl(ctx, n, res.File); fn != nil {
					checkTypeReferenceOnNode(o.filename, fn, ft, ctx, o.guards, o.level0)
				}
			case syntax.KindClosureExpr:
				if fn := memoLowerExpr(ctx, n, res.File); fn != nil {
					checkTypeReferenceOnNode(o.filename, fn, ft, ctx, o.guards, o.level0)
				}
			case syntax.KindPropertyDecl:
				for _, p := range memoLowerPropertyDecl(ctx, n, res.File) {
					checkTypeReferenceOnNode(o.filename, p, ft, ctx, o.guards, o.level0)
				}
			case syntax.KindConstDecl:
				for _, c := range syntax.LowerClassConstDeclNode(n, res.File) {
					checkTypeReferenceOnNode(o.filename, c, ft, ctx, o.guards, o.level0)
				}
			case syntax.KindCatchClause:
				if catch := syntax.LowerCatchClauseNode(n, res.File); catch != nil {
					checkTypeReferenceOnNode(o.filename, catch, ft, ctx, o.guards, o.level0)
				}
			case syntax.KindAttribute:
				if !inStatementBody {
					if attr := syntax.LowerAttributeNode(n, res.File); attr != nil {
						checkTypeReferenceOnNode(o.filename, attr, ft, ctx, o.guards, o.level0)
					}
				}
			}
		}

		if runSymbols && !symbolSuppressed[cstKeyOf(n)] {
			switch n.Kind() {
			case syntax.KindCallExpr:
				n.ForEachChildDesc(func(green *syntax.GreenNode, offset int) bool {
					switch green.Kind() {
					case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr, syntax.KindStaticMemberAccessExpr:
						callCallees[cstRedNodeKey{green: green, offset: offset}] = true
					}
					return true
				})
				node := memoLowerExpr(ctx, n, res.File)
				if node == nil {
					syntax.Walk(n, func(d *syntax.RedNode) bool {
						symbolSuppressed[cstKeyOf(d)] = true
						return true
					})
					break
				}
				checkSymbolOnNode(o.filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, o.guards, o.level0)
				if fc, ok := node.(*ast.FunctionCallNode); ok {
					if cf, ok := fc.Name.(*ast.ClassConstFetchNode); ok {
						checkSymbolOnNode(o.filename, cf, classCtx(class), fnCtx(currentFn), ft, ctx, o.guards, o.level0)
					}
				}
			case syntax.KindNewExpr:
				if node := memoLowerExpr(ctx, n, res.File); node != nil {
					checkSymbolOnNode(o.filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, o.guards, o.level0)
				}
			case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr, syntax.KindStaticMemberAccessExpr:
				if !callCallees[cstKeyOf(n)] {
					if node := memoLowerExpr(ctx, n, res.File); node != nil {
						checkSymbolOnNode(o.filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, o.guards, o.level0)
					}
				}
			}
		}

		if !o.collectStructural {
			return
		}

		if n.Kind() == syntax.KindCallExpr && !visibilitySuppressed[cstKeyOf(n)] {
			node := memoLowerExpr(ctx, n, res.File)
			if node == nil {
				syntax.Walk(n, func(d *syntax.RedNode) bool {
					visibilitySuppressed[cstKeyOf(d)] = true
					return true
				})
			} else {
				appendMethodVisibilityOnNode(o.filename, node, classCtx(class), fnCtx(currentFn), ft, ctx, o.methodVisibility)
			}
		}

		switch n.Kind() {
		case syntax.KindThrowStmt:
			if node := syntax.LowerStmtNode(n, res.File); node != nil {
				appendThrowTypeOnNode(o.filename, node, ft, ctx, o.throwType)
			}
		case syntax.KindThrowExpr:
			if node := memoLowerExpr(ctx, n, res.File); node != nil {
				appendThrowTypeOnNode(o.filename, node, ft, ctx, o.throwType)
			}
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			if class != nil && class.Kind() == syntax.KindInterfaceDecl {
				if im := memoLowerInterfaceMethod(ctx, n, res.File); im != nil {
					appendPHPDocIssuesOnNode(o.filename, im, lowerClass(class), ft, ctx, o.phpDoc)
					if o.collectMissingTypes {
						appendMissingTypeIssuesOnNode(o.filename, im, lowerClass(class), ft, ctx, o.missingType)
					}
					if o.collectReturn {
						appendReturnTypeOnNode(o.filename, im, lowerClass(class), ft, ctx, o.returnType)
					}
				}
				return
			}
			if fn := memoLowerFunctionDecl(ctx, n, res.File); fn != nil {
				appendPHPDocIssuesOnNode(o.filename, fn, lowerClass(class), ft, ctx, o.phpDoc)
				if o.collectMissingTypes {
					appendMissingTypeIssuesOnNode(o.filename, fn, lowerClass(class), ft, ctx, o.missingType)
				}
				if o.collectReturn {
					appendReturnTypeOnNode(o.filename, fn, lowerClass(class), ft, ctx, o.returnType)
				}
			}
		case syntax.KindClosureExpr:
			if fn := memoLowerExpr(ctx, n, res.File); fn != nil {
				appendPHPDocIssuesOnNode(o.filename, fn, lowerClass(class), ft, ctx, o.phpDoc)
				if o.collectMissingTypes {
					appendMissingTypeIssuesOnNode(o.filename, fn, lowerClass(class), ft, ctx, o.missingType)
				}
				if o.collectReturn {
					appendReturnTypeOnNode(o.filename, fn, lowerClass(class), ft, ctx, o.returnType)
				}
			}
		case syntax.KindPropertyDecl:
			for _, p := range memoLowerPropertyDecl(ctx, n, res.File) {
				appendPHPDocIssuesOnNode(o.filename, p, lowerClass(class), ft, ctx, o.phpDoc)
				if o.collectMissingTypes {
					appendMissingTypeIssuesOnNode(o.filename, p, lowerClass(class), ft, ctx, o.missingType)
				}
			}
		}
	})
}
