package analyse

import "github.com/ayanozturk/go-php-parser/syntax"

// walkSyntaxConfigured is the CST-direct analogue of walkAllConfigured
// (phpstan_level0_walk.go): a pre-order traversal over *syntax.RedNode that
// threads the enclosing class/function declaration and a FileTypeContext
// down to fn, mirroring the ast.Node walker's context-propagation rules for
// the boundary kinds it special-cases (class-likes, functions/methods,
// closures). This is Phase 1 of the CST-direct migration (see AGENTS.md
// "CST-direct migration" and /memories/repo/cst-direct-migration.md) — an
// additive skeleton, not yet wired into any production rule or fn's
// FileTypeContext (that's Phase 2: a CST-direct CollectFileTypeContext).
//
// Unlike walkAllConfigured, which needs one type-switch case per concrete
// ast.Node type to know which fields to recurse into, the CST's uniform
// Children() means only two things need special-casing here: which kinds
// are pure grouping containers with no ast.Node equivalent (so fn should
// never be invoked for them), and which kinds open a new class/function
// scope for context-propagation purposes. Every other kind is walked
// generically.
func walkSyntaxConfigured(root *syntax.RedNode, ft FileTypeContext, fn func(n, class, currentFn *syntax.RedNode, ft FileTypeContext)) {
	if root == nil {
		return
	}
	var walk func(n, class, currentFn *syntax.RedNode, ft FileTypeContext)
	walk = func(n, class, currentFn *syntax.RedNode, ft FileTypeContext) {
		if n == nil {
			return
		}
		if !syntaxContainerOnlyKinds[n.Kind()] {
			fn(n, class, currentFn, ft)
		}
		nextClass, nextFn := class, currentFn
		switch n.Kind() {
		case syntax.KindClassDecl, syntax.KindInterfaceDecl, syntax.KindTraitDecl,
			syntax.KindEnumDecl, syntax.KindAnonymousClass:
			nextClass = n
		case syntax.KindFunctionDecl, syntax.KindMethodDecl,
			syntax.KindClosureExpr, syntax.KindArrowFunctionExpr:
			nextFn = n
		}
		for _, c := range n.Children() {
			walk(c, nextClass, nextFn, ft)
		}
	}
	walk(root, nil, nil, ft)
}

// syntaxContainerOnlyKinds are CST kinds that exist purely for grouping
// (param lists, member lists, statement lists, clause wrappers, …) and have
// no ast.Node equivalent — walkSyntaxConfigured recurses through them but
// never invokes fn, mirroring how walkAllConfigured never calls its walk
// callback on things like an else-if clause or a raw statement slice, since
// those aren't ast.Node values in the classic model.
var syntaxContainerOnlyKinds = map[syntax.Kind]bool{
	syntax.KindToken:               true,
	syntax.KindTokenList:           true,
	syntax.KindMissing:             true,
	syntax.KindError:               true,
	syntax.KindFile:                true,
	syntax.KindModifierList:        true,
	syntax.KindParamList:           true,
	syntax.KindCallableParamList:   true,
	syntax.KindMemberList:          true,
	syntax.KindArgList:             true,
	syntax.KindNameList:            true,
	syntax.KindStatementList:       true,
	syntax.KindAttributeList:       true,
	syntax.KindExtendsClause:       true,
	syntax.KindImplementsClause:    true,
	syntax.KindUseTraitClause:      true,
	syntax.KindUseGroup:            true,
	syntax.KindElseIfClause:        true,
	syntax.KindElseClause:          true,
	syntax.KindTraitAdaptationList: true,
	syntax.KindPropertyHookList:    true,
	syntax.KindClosureUseClause:    true,
}
