package ast

import (
	"testing"
)

func TestPropertyFetchNode_Basic(t *testing.T) {
	obj := &VariableNode{Name: "this", Pos: Position{Line: 1, Column: 2}}
	pf := &PropertyFetchNode{
		Object:   obj,
		Property: "name",
		Pos:      Position{Line: 1, Column: 5},
	}

	if pf.NodeType() != "PropertyFetch" {
		t.Errorf("NodeType() = %q, want %q", pf.NodeType(), "PropertyFetch")
	}
	if pf.GetPos() != (Position{Line: 1, Column: 5}) {
		t.Errorf("GetPos() = %+v, want %+v", pf.GetPos(), Position{Line: 1, Column: 5})
	}
	pf.SetPos(Position{Line: 2, Column: 3})
	if pf.GetPos() != (Position{Line: 2, Column: 3}) {
		t.Errorf("SetPos() did not update position")
	}
	str := pf.String()
	if str == "" || str == "PropertyFetch()" {
		t.Errorf("String() = %q, want non-empty description", str)
	}
	if pf.TokenLiteral() != "name" {
		t.Errorf("TokenLiteral() = %q, want %q", pf.TokenLiteral(), "name")
	}
}
