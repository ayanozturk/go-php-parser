package ast

// TypeText returns the source spelling of a native type node, or "" if nil.
// Prefer this over ad-hoc TokenLiteral checks so union/intersection/identifier
// types share one display path while the lossless syntax cutover completes.
func TypeText(n Node) string {
	if n == nil {
		return ""
	}
	switch t := n.(type) {
	case *UnionTypeNode:
		return t.TokenLiteral()
	case *IntersectionTypeNode:
		return t.TokenLiteral()
	case *NullableTypeNode:
		return t.TokenLiteral()
	case *ParenthesizedTypeNode:
		return t.TokenLiteral()
	case *CallableTypeNode:
		return t.TokenLiteral()
	case *IdentifierNode:
		return t.Value
	default:
		return n.TokenLiteral()
	}
}
