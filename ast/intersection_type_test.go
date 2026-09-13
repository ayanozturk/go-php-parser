package ast

import (
	"testing"
)

func TestIntersectionTypeNodeBasic(t *testing.T) {
	types := []Node{
		&IdentifierNode{Value: "A"},
		&IdentifierNode{Value: "B"},
		&IdentifierNode{Value: "C"},
	}
	pos := Position{Line: 7, Column: 3}
	n := &IntersectionTypeNode{Types: types, Pos: pos}

	if n.NodeType() != "IntersectionType" {
		t.Errorf("NodeType() = %s; want IntersectionType", n.NodeType())
	}
	if n.GetPos() != pos {
		t.Errorf("GetPos() = %+v; want %+v", n.GetPos(), pos)
	}
	n.SetPos(Position{Line: 8, Column: 4})
	if n.GetPos() != (Position{Line: 8, Column: 4}) {
		t.Errorf("SetPos() failed, got %+v", n.GetPos())
	}
	if n.TokenLiteral() != "A&B&C" {
		t.Errorf("TokenLiteral() = %s; want A&B&C", n.TokenLiteral())
	}
	if n.String() != "A & B & C" {
		t.Errorf("String() = %s; want A & B & C", n.String())
	}
	if len(n.Types) != 3 || TypeText(n.Types[0]) != "A" || TypeText(n.Types[2]) != "C" {
		t.Errorf("Types = %+v; want A B C nodes", n.Types)
	}
}
