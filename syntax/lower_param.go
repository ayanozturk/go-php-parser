package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func lowerParamList(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	list := n
	if n.Kind() != KindParamList {
		list = n.FirstChildOfKind(KindParamList)
	}
	if list == nil {
		return nil
	}
	var out []ast.Node
	for _, c := range list.Children() {
		if c.Kind() != KindParam {
			continue
		}
		if p := lowerParam(c, file); p != nil {
			out = append(out, p)
		}
	}
	return out
}

func lowerParam(n *RedNode, file *File) *ast.ParamNode {
	if n == nil || n.Kind() != KindParam {
		return nil
	}
	pos, end := nodePos(file, n)
	p := &ast.ParamNode{
		Pos:    pos,
		EndPos: end,
		PHPDoc: leadingDocFromNode(n),
	}
	p.Modifiers = lowerModifiers(n)
	if len(p.Modifiers) > 0 && p.Modifiers.Visibility() != "" {
		p.IsPromoted = true
	}
	if p.Modifiers.HasName("readonly") {
		p.IsReadonly = true
	}
	for _, c := range n.Children() {
		switch {
		case isTypeKind(c.Kind()):
			p.TypeHint = lowerType(c, file)
		case isTokenType(c, token.T_AMPERSAND):
			p.IsByRef = true
		case isTokenType(c, token.T_ELLIPSIS):
			p.IsVariadic = true
		case isTokenType(c, token.T_VARIABLE):
			p.Name = stripVarDollar(tokenLiteral(c))
		}
	}
	return p
}
