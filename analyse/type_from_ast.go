package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

// TypeFromAST builds analyse.Type from a structured AST type node.
// Prefer this over ParseType(ast.TypeText(n)) at AST edges so unions,
// intersections, nullable, and callable shapes stay structural.
func TypeFromAST(n ast.Node, typeCtx FileTypeContext) Type {
	if n == nil {
		return EmptyType()
	}
	switch t := n.(type) {
	case *ast.IdentifierNode:
		return ParseType(normalizeTypeWithContext(t.Value, typeCtx))
	case *ast.NullableTypeNode:
		return nullableType(TypeFromAST(t.Inner, typeCtx))
	case *ast.ParenthesizedTypeNode:
		return TypeFromAST(t.Inner, typeCtx)
	case *ast.UnionTypeNode:
		parts := make([]Type, 0, len(t.Types))
		for _, child := range t.Types {
			parts = append(parts, TypeFromAST(child, typeCtx))
		}
		return unionTypes(parts...)
	case *ast.IntersectionTypeNode:
		parts := make([]Type, 0, len(t.Types))
		for _, child := range t.Types {
			parts = append(parts, TypeFromAST(child, typeCtx))
		}
		return intersectionTypes(parts...)
	case *ast.CallableTypeNode:
		return ParseType("callable")
	default:
		text := strings.TrimSpace(ast.TypeText(n))
		if text == "" {
			return EmptyType()
		}
		return ParseType(normalizeTypeWithContext(text, typeCtx))
	}
}
