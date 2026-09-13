package ast

import "testing"

func TestCallableTypeNodeTokenLiteral(t *testing.T) {
	bare := &CallableTypeNode{HasSignature: false}
	if bare.TokenLiteral() != "callable" {
		t.Errorf("bare callable TokenLiteral: got %q", bare.TokenLiteral())
	}

	signed := &CallableTypeNode{
		HasSignature: true,
		Params: []*CallableParamNode{
			{
				TypeHint: &IdentifierNode{Value: "Foo"},
				Name:     "x",
			},
		},
		ReturnType: &IdentifierNode{Value: "Bar"},
	}
	if signed.TokenLiteral() != "callable(Foo$x):Bar" {
		t.Errorf("signed callable TokenLiteral: got %q", signed.TokenLiteral())
	}
}
