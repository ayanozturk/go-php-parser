package ast

import (
	"testing"
)

func TestClassNodeMethods(t *testing.T) {
	c := &ClassNode{
		Name:       "MyClass",
		Extends:    "BaseClass",
		Implements: []string{"Iface1", "Iface2"},
		Pos:        Position{Line: 10, Column: 5},
	}
	if c.NodeType() != "Class" {
		t.Errorf("NodeType: got %q", c.NodeType())
	}
	if c.GetPos().Line != 10 || c.GetPos().Column != 5 {
		t.Errorf("GetPos: got %+v", c.GetPos())
	}
	c.SetPos(Position{Line: 20, Column: 8})
	if c.GetPos().Line != 20 || c.GetPos().Column != 8 {
		t.Errorf("SetPos: got %+v", c.GetPos())
	}
	str := c.String()
	if str == "" || str == "Class(MyClass) @ 20:8" {
		t.Errorf("String: got %q", str)
	}
	if c.TokenLiteral() != "class" {
		t.Errorf("TokenLiteral: got %q", c.TokenLiteral())
	}
}

func TestPropertyNodeMethods(t *testing.T) {
	p := &PropertyNode{
		Name:       "foo",
		Visibility: "private",
		Pos:        Position{Line: 2, Column: 3},
	}
	if p.NodeType() != "Property" {
		t.Errorf("NodeType: got %q", p.NodeType())
	}
	if p.GetPos().Line != 2 || p.GetPos().Column != 3 {
		t.Errorf("GetPos: got %+v", p.GetPos())
	}
	p.SetPos(Position{Line: 4, Column: 5})
	if p.GetPos().Line != 4 || p.GetPos().Column != 5 {
		t.Errorf("SetPos: got %+v", p.GetPos())
	}
	str := p.String()
	if str == "" || str == "Property($foo) @ 4:5" {
		t.Errorf("String: got %q", str)
	}
	if p.TokenLiteral() != "foo" {
		t.Errorf("TokenLiteral: got %q", p.TokenLiteral())
	}
}
