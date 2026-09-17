package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CollectFileTypeContextFromSyntax is the CST-direct analogue of
// CollectFileTypeContext (name_resolver.go): it derives the same Namespace/
// Aliases/FunctionAliases/ConstAliases/Classes fields directly from
// *syntax.RedNode, without lowering to []ast.Node first. This is Phase 2 of
// the CST-direct migration (see AGENTS.md "CST-direct migration" and
// /memories/repo/cst-direct-migration.md) — additive, not yet wired into any
// production rule or the registry.
//
// Two fields are deliberately left unpopulated (nil maps), same shape as the
// ast.Node path returns but empty, rather than guessed at:
//   - ClassNodes: keyed by *ast.ClassNode, which a pure CST walk has no
//     equivalent for. Consumers needing enclosing-class member lookups
//     (return_type_rule.go's generic-parent resolution, context.go) aren't
//     CST-direct yet; porting them is later-phase work.
//   - Constants: top-level literal `const` value tracking. It has exactly
//     one consumer (return_type_rule.go's array-key inference) and would
//     require re-implementing PHP string-literal unescaping against raw
//     CST tokens for zero currently-exercised benefit here.
func CollectFileTypeContextFromSyntax(root *syntax.RedNode) FileTypeContext {
	ctx := FileTypeContext{
		Aliases:         make(map[string]string),
		FunctionAliases: make(map[string]string),
		ConstAliases:    make(map[string]string),
		Classes:         make(map[string]ResolvedClass),
		ClassNodes:      make(map[string]*ast.ClassNode),
		Constants:       make(map[string]string),
	}
	if root == nil {
		return ctx
	}
	collectSyntaxFileTypeContext(root.Children(), "", &ctx)
	return ctx
}

func collectSyntaxFileTypeContext(children []*syntax.RedNode, currentNS string, ctx *FileTypeContext) {
	namespace := currentNS
	for i := 0; i < len(children); i++ {
		n := children[i]
		switch n.Kind() {
		case syntax.KindNamespaceDecl:
			nsName := syntaxNamespaceDeclName(n)
			body, consumed := syntax.NamespaceBody(n, children, i)
			if len(body) > 0 {
				if ctx.Namespace == "" {
					ctx.Namespace = nsName
				}
				collectSyntaxFileTypeContext(body, nsName, ctx)
				i += consumed
				continue
			}
			namespace = nsName
			if ctx.Namespace == "" {
				ctx.Namespace = nsName
			}
		case syntax.KindUseDecl:
			syntax.AppendTypedUseAliases(n, ctx.Aliases, ctx.FunctionAliases, ctx.ConstAliases)
		case syntax.KindClassDecl:
			registerSyntaxClass(n, namespace, ctx)
		case syntax.KindInterfaceDecl:
			registerSyntaxInterface(n, namespace, ctx)
		}
	}
	if ctx.Namespace == "" {
		ctx.Namespace = namespace
	}
}

func syntaxNamespaceDeclName(n *syntax.RedNode) string {
	for _, c := range n.Children() {
		if syntax.IsNameKind(c.Kind()) {
			return strings.TrimPrefix(syntax.NameText(c), `\`)
		}
	}
	return ""
}

func syntaxClassLikeNameAndClauses(n *syntax.RedNode) (name *syntax.RedNode, extends, implements *syntax.RedNode) {
	for _, c := range n.Children() {
		switch {
		case name == nil && syntax.IsNameKind(c.Kind()):
			name = c
		case c.Kind() == syntax.KindExtendsClause:
			extends = c
		case c.Kind() == syntax.KindImplementsClause:
			implements = c
		}
	}
	return name, extends, implements
}

func registerSyntaxClass(n *syntax.RedNode, namespace string, ctx *FileTypeContext) {
	nameNode, extendsClause, implementsClause := syntaxClassLikeNameAndClauses(n)
	if nameNode == nil {
		return
	}
	className := resolveClassLikeInContext(namespace, ctx.Aliases, syntax.UnqualifiedTail(syntax.NameText(nameNode)))
	resolved := ResolvedClass{Name: className}
	if names := syntax.ClauseNames(extendsClause); len(names) > 0 {
		resolved.Extends = []string{resolveClassLikeInContext(namespace, ctx.Aliases, names[0])}
	}
	if names := syntax.ClauseNames(implementsClause); len(names) > 0 {
		resolved.Implements = make([]string, 0, len(names))
		for _, name := range names {
			resolved.Implements = append(resolved.Implements, resolveClassLikeInContext(namespace, ctx.Aliases, name))
		}
	}
	ctx.Classes[asciiLowerIdent(strings.TrimPrefix(className, `\`))] = resolved
}

func registerSyntaxInterface(n *syntax.RedNode, namespace string, ctx *FileTypeContext) {
	nameNode, extendsClause, _ := syntaxClassLikeNameAndClauses(n)
	if nameNode == nil {
		return
	}
	interfaceName := resolveClassLikeInContext(namespace, ctx.Aliases, syntax.UnqualifiedTail(syntax.NameText(nameNode)))
	resolved := ResolvedClass{Name: interfaceName}
	if names := syntax.ClauseNames(extendsClause); len(names) > 0 {
		resolved.Extends = make([]string, 0, len(names))
		for _, name := range names {
			resolved.Extends = append(resolved.Extends, resolveClassLikeInContext(namespace, ctx.Aliases, name))
		}
	}
	ctx.Classes[asciiLowerIdent(strings.TrimPrefix(interfaceName, `\`))] = resolved
}
