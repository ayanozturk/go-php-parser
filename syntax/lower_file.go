package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
)

// LowerFile lowers KindFile children to top-level classic AST nodes.
// Exported for the syntax/lower facade; prefer syntax.ParseASTForIndex.
// Gaps: attributes, anonymous class, closures/arrows, full expr/stmt lowering;
// method bodies stay empty in index mode. Enum case values stay nil.

// File lowers KindFile children to top-level classic AST nodes.
func LowerFile(root *RedNode, file *File) []ast.Node {
	if root == nil {
		return nil
	}
	if root.Kind() != KindFile {
		nodes, _ := lowerTopLevel(root, file)
		return nodes
	}
	children := root.Children()
	var out []ast.Node
	for i := 0; i < len(children); i++ {
		c := children[i]
		if c.Kind() == KindNamespaceDecl {
			ns, consumed := lowerNamespace(c, file, children, i)
			if ns != nil {
				out = append(out, ns)
			}
			i += consumed
			continue
		}
		nodes, _ := lowerTopLevel(c, file)
		out = append(out, nodes...)
	}
	return out
}

// lowerTopLevel lowers one red node that may appear at file or namespace scope.
// ok is false when the kind is intentionally skipped (unsupported / trivia).
func lowerTopLevel(n *RedNode, file *File) (nodes []ast.Node, ok bool) {
	if n == nil {
		return nil, false
	}
	switch n.Kind() {
	case KindUseDecl:
		return lowerUseDecl(n, file), true
	case KindClassDecl:
		if cls := lowerClass(n, file); cls != nil {
			return []ast.Node{cls}, true
		}
		return nil, true
	case KindInterfaceDecl:
		if iface := lowerInterface(n, file); iface != nil {
			return []ast.Node{iface}, true
		}
		return nil, true
	case KindFunctionDecl, KindMethodDecl:
		if fn := lowerFunction(n, file); fn != nil {
			return []ast.Node{fn}, true
		}
		return nil, true
	case KindConstDecl:
		// Global const — name-only MVP via ClassConst-like children.
		return lowerClassConsts(n, file), true
	case KindTraitDecl:
		if tr := lowerTrait(n, file); tr != nil {
			return []ast.Node{tr}, true
		}
		return nil, true
	case KindEnumDecl:
		if en := lowerEnum(n, file); en != nil {
			return []ast.Node{en}, true
		}
		return nil, true
	case KindAnonymousClass:
		// Standalone anonymous class is unexpected at top level; skip.
		return nil, true
	case KindToken, KindTokenList, KindError, KindMissing,
		KindEmptyStmt, KindAttributeList:
		return nil, false
	default:
		// File-scope statements / expressions (ExpressionStmt, Throw, Echo, …).
		if isStmtKind(n.Kind()) {
			if stmt := lowerStmt(n, file); stmt != nil {
				return []ast.Node{stmt}, true
			}
			return nil, true
		}
		if isExprKind(n.Kind()) || isNameKind(n.Kind()) {
			if e := lowerExpr(n, file); e != nil {
				pos, end := nodePos(file, n)
				return []ast.Node{&ast.ExpressionStmt{Expr: e, Pos: pos, EndPos: end}}, true
			}
			return nil, true
		}
		return nil, false
	}
}
