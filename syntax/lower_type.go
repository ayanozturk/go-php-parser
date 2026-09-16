package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

func lowerType(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	switch n.Kind() {
	case KindNullableType:
		var inner ast.Node
		for _, c := range n.Children() {
			if c.Kind() == KindToken {
				continue
			}
			inner = lowerType(c, file)
			break
		}
		return &ast.NullableTypeNode{Inner: inner, Pos: pos, EndPos: end}
	case KindUnionType:
		return &ast.UnionTypeNode{Types: lowerTypeParts(n, file), Pos: pos, EndPos: end}
	case KindIntersectionType:
		return &ast.IntersectionTypeNode{Types: lowerTypeParts(n, file), Pos: pos, EndPos: end}
	case KindParenthesizedType:
		var inner ast.Node
		for _, c := range n.Children() {
			if c.Kind() == KindToken {
				continue
			}
			inner = lowerType(c, file)
			break
		}
		return &ast.ParenthesizedTypeNode{Inner: inner, Pos: pos, EndPos: end}
	case KindNamedType:
		name := firstNameChild(n)
		val := NameText(name)
		if val == "" {
			val = strings.TrimSpace(TypeText(n))
		}
		np, ne := pos, end
		if name != nil {
			np, ne = nodePos(file, name)
		}
		return &ast.IdentifierNode{Value: val, Pos: np, EndPos: ne}
	case KindPrimitiveType:
		val := strings.TrimSpace(TypeText(n))
		return &ast.IdentifierNode{Value: val, Pos: pos, EndPos: end}
	case KindCallableType:
		// MVP: surface as identifier "callable" / full TypeText for signatures.
		val := strings.TrimSpace(TypeText(n))
		if val == "" {
			val = "callable"
		}
		return &ast.IdentifierNode{Value: val, Pos: pos, EndPos: end}
	default:
		if isNameKind(n.Kind()) {
			val := NameText(n)
			return &ast.IdentifierNode{Value: val, Pos: pos, EndPos: end}
		}
		if text := strings.TrimSpace(TypeText(n)); text != "" {
			return &ast.IdentifierNode{Value: text, Pos: pos, EndPos: end}
		}
		return nil
	}
}

func lowerTypeParts(n *RedNode, file *File) []ast.Node {
	var parts []ast.Node
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if t := lowerType(c, file); t != nil {
			parts = append(parts, t)
		}
	}
	return parts
}

func isTypeKind(k Kind) bool {
	switch k {
	case KindNamedType, KindPrimitiveType, KindNullableType,
		KindUnionType, KindIntersectionType, KindParenthesizedType,
		KindCallableType:
		return true
	default:
		return false
	}
}
