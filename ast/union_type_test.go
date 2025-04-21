package ast

import "testing"

func TestUnionTypeNodeMethods(t *testing.T) {
	u := &UnionTypeNode{
		Types: []string{"int", "string"},
		Pos:   Position{Line: 3, Column: 4},
	}
	if u.NodeType() != "UnionType" {
		t.Errorf("NodeType: got %q", u.NodeType())
	}
	if u.GetPos().Line != 3 || u.GetPos().Column != 4 {
		t.Errorf("GetPos: got %+v", u.GetPos())
	}
	u.SetPos(Position{Line: 5, Column: 6})
	if u.GetPos().Line != 5 || u.GetPos().Column != 6 {
		t.Errorf("SetPos: got %+v", u.GetPos())
	}
	str := u.String()
	if str != "UnionType(int|string) @ 5:6" {
		t.Errorf("String: got %q", str)
	}
	if u.TokenLiteral() != "int|string" {
		t.Errorf("TokenLiteral: got %q", u.TokenLiteral())
	}
}
