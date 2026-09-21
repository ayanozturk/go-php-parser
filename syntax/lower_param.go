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
	list.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindParam {
			return true
		}
		param := RedNode{File: list.File, Green: green, Offset: offset}
		if p := lowerParam(&param, file); p != nil {
			out = append(out, p)
		}
		return true
	})
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
	seenAssign := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch {
		case k == KindAttributeList:
			attrList := RedNode{File: n.File, Green: green, Offset: offset}
			p.Attributes = append(p.Attributes, lowerAttributeList(&attrList, file)...)
		case isTypeKind(k):
			typeNode := RedNode{File: n.File, Green: green, Offset: offset}
			p.TypeHint = lowerType(&typeNode, file)
		case k == KindToken && green.TokenType() == token.T_AMPERSAND:
			p.IsByRef = true
		case k == KindToken && green.TokenType() == token.T_ELLIPSIS:
			p.IsVariadic = true
		case k == KindToken && green.TokenType() == token.T_VARIABLE:
			p.Name = stripVarDollar(greenTokenLiteral(green))
		case k == KindToken && green.TokenType() == token.T_ASSIGN:
			seenAssign = true
		case seenAssign && (isExprKind(k) || isNameKind(k)):
			p.DefaultValue = lowerExprAt(file, green, offset)
			seenAssign = false
		}
		return true
	})
	return p
}
