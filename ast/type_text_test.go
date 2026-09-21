package ast

import "testing"

func TestTypeText(t *testing.T) {
	id := &IdentifierNode{Value: "string"}
	union := &UnionTypeNode{Types: []Node{&IdentifierNode{Value: "int"}, &IdentifierNode{Value: "string"}}}
	inter := &IntersectionTypeNode{Types: []Node{&IdentifierNode{Value: "A"}, &IdentifierNode{Value: "B"}}}
	nullable := &NullableTypeNode{Inner: id}
	paren := &ParenthesizedTypeNode{Inner: inter}
	callable := &CallableTypeNode{HasSignature: false}
	other := &IntegerLiteral{Value: 7}

	cases := []struct {
		name string
		node Node
		want string
	}{
		{"nil", nil, ""},
		{"identifier", id, "string"},
		{"union", union, union.TokenLiteral()},
		{"intersection", inter, inter.TokenLiteral()},
		{"nullable", nullable, "?string"},
		{"parenthesized", paren, paren.TokenLiteral()},
		{"callable", callable, "callable"},
		{"default", other, other.TokenLiteral()},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := TypeText(tt.node); got != tt.want {
				t.Fatalf("TypeText() = %q, want %q", got, tt.want)
			}
		})
	}
}
