package ast

import (
	"reflect"
	"testing"
)

func TestIntersectionTypeNode_Basic(t *testing.T) {
	types := []string{"A", "B", "C"}
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
	if n.TokenLiteral() != "&" {
		t.Errorf("TokenLiteral() = %s; want &", n.TokenLiteral())
	}
	if n.String() != "A & B & C" {
		t.Errorf("String() = %s; want A & B & C", n.String())
	}
	if !reflect.DeepEqual(n.Types, types) {
		t.Errorf("Types = %+v; want %+v", n.Types, types)
	}
}
