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

func TestCallableParamAndTypeStringBranches(t *testing.T) {
	typeOnly := &CallableParamNode{TypeHint: &IdentifierNode{Value: "int"}, Pos: Position{Line: 1, Column: 2}}
	if got := typeOnly.TokenLiteral(); got != "int" {
		t.Fatalf("type-only TokenLiteral=%q", got)
	}
	if got := typeOnly.String(); got != "CallableParam(int) @ 1:2" {
		t.Fatalf("type-only String=%q", got)
	}

	nameOnly := &CallableParamNode{Name: "n", Pos: Position{Line: 2, Column: 3}}
	if got := nameOnly.TokenLiteral(); got != "$n" {
		t.Fatalf("name-only TokenLiteral=%q", got)
	}
	if got := nameOnly.String(); got != "CallableParam($n) @ 2:3" {
		t.Fatalf("name-only String=%q", got)
	}

	both := &CallableParamNode{TypeHint: &IdentifierNode{Value: "Foo"}, Name: "x", Pos: Position{Line: 3, Column: 4}}
	if got := both.TokenLiteral(); got != "Foo$x" {
		t.Fatalf("both TokenLiteral=%q", got)
	}

	multi := &CallableTypeNode{
		HasSignature: true,
		Params: []*CallableParamNode{
			{TypeHint: &IdentifierNode{Value: "A"}},
			{Name: "b"},
		},
		Pos: Position{Line: 4, Column: 5},
	}
	if got := multi.TokenLiteral(); got != "callable(A,$b)" {
		t.Fatalf("multi TokenLiteral=%q", got)
	}
	if got := multi.String(); got != "CallableType(callable(A,$b)) @ 4:5" {
		t.Fatalf("multi String=%q", got)
	}
	if got := multi.NodeType(); got != "CallableType" {
		t.Fatalf("NodeType=%q", got)
	}
	assertLoopControlSpan(t, multi, Position{Line: 4, Column: 5}, Position{})
}
