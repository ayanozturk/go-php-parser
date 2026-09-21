package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
)

// LowerFile lowers KindFile children to top-level classic AST nodes.
// Exported for the syntax/lower facade; prefer syntax.ParseASTForIndex.
// Method/function bodies stay empty in index mode (KindTokenList).
// Property-hook Expr/Body and property DefaultValue lower whenever CST has them.

type topLevelChild struct {
	green  *GreenNode
	offset int
}

// File lowers KindFile children to top-level classic AST nodes.
func LowerFile(root *RedNode, file *File) []ast.Node {
	if root == nil {
		return nil
	}
	if root.Kind() != KindFile {
		nodes, _ := lowerTopLevel(root, file)
		return nodes
	}
	var children []topLevelChild
	root.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindToken {
			return true
		}
		children = append(children, topLevelChild{green: green, offset: offset})
		return true
	})
	var out []ast.Node
	for i := 0; i < len(children); i++ {
		ch := children[i]
		if ch.green.Kind() == KindNamespaceDecl {
			ns, consumed := lowerNamespaceAt(file, ch, children, i)
			if ns != nil {
				out = append(out, ns)
			}
			i += consumed
			continue
		}
		withPooledRed(file, root, ch.green, ch.offset, func(c *RedNode) {
			nodes, _ := lowerTopLevel(c, file)
			out = append(out, nodes...)
		})
	}
	return out
}

func lowerNamespaceAt(file *File, n topLevelChild, siblings []topLevelChild, idx int) (*ast.NamespaceNode, int) {
	var ns *ast.NamespaceNode
	var consumed int
	withPooledRed(file, nil, n.green, n.offset, func(red *RedNode) {
		ns, consumed = lowerNamespace(red, file, siblings, idx)
	})
	return ns, consumed
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
	case KindEmptyStmt:
		if stmt := lowerStmt(n, file); stmt != nil {
			return []ast.Node{stmt}, true
		}
		return nil, true
	case KindAttributeList:
		return lowerAttributeList(n, file), true
	case KindToken, KindTokenList, KindError, KindMissing:
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

func lowerAttributeList(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	var out []ast.Node
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindAttributeGroup {
			return true
		}
		forEachChildDescGreen(green, offset, func(attrGreen *GreenNode, attrOff int) bool {
			if attrGreen.Kind() != KindAttribute {
				return true
			}
			withPooledRed(file, n, attrGreen, attrOff, func(attr *RedNode) {
				if node := lowerAttribute(attr, file); node != nil {
					out = append(out, node)
				}
			})
			return true
		})
		return true
	})
	return out
}

func lowerAttribute(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var name string
	var argsGreen *GreenNode
	var argsOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindArgList {
			argsGreen, argsOff = green, offset
			return true
		}
		if isNameKind(k) && name == "" {
			name = nameTextAt(file, green, offset)
		}
		return true
	})
	if name == "" {
		return nil
	}
	argNodes := lowerArgListFromGreen(file, argsGreen, argsOff)
	return &ast.AttributeNode{
		Name:      name,
		Arguments: argNodes,
		Pos:       pos,
		EndPos:    end,
	}
}
