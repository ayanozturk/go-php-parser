package ast

import (
	"reflect"
	"testing"
)

func TestConstantNode_Basic(t *testing.T) {
	val := &IntegerLiteral{Value: 123}
	pos := Position{Line: 10, Column: 5}
	n := &ConstantNode{
		Name:       "FOO",
		Type:       "int",
		Visibility: "public",
		Value:      val,
		Pos:        pos,
	}

	if n.NodeType() != "Constant" {
		t.Errorf("NodeType() = %s; want Constant", n.NodeType())
	}
	if n.GetPos() != pos {
		t.Errorf("GetPos() = %+v; want %+v", n.GetPos(), pos)
	}
	n.SetPos(Position{Line: 11, Column: 6})
	if n.GetPos() != (Position{Line: 11, Column: 6}) {
		t.Errorf("SetPos() failed, got %+v", n.GetPos())
	}
	if n.TokenLiteral() != "FOO" {
		t.Errorf("TokenLiteral() = %s; want FOO", n.TokenLiteral())
	}
	if n.String() == "" {
		t.Error("String() should not be empty")
	}
	if n.Name != "FOO" {
		t.Errorf("Name = %s; want FOO", n.Name)
	}
	if n.Type != "int" {
		t.Errorf("Type = %s; want int", n.Type)
	}
	if n.Visibility != "public" {
		t.Errorf("Visibility = %s; want public", n.Visibility)
	}
	if !reflect.DeepEqual(n.Value, val) {
		t.Errorf("Value = %+v; want %+v", n.Value, val)
	}
}
