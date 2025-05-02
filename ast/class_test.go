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
func TestNewNode(t *testing.T) {
	args := []Node{&StringLiteral{Value: "arg1", Pos: Position{Line: 1, Column: 1}}}
	n := &NewNode{ClassName: "MyClass", Args: args, Pos: Position{Line: 2, Column: 2}}
	if n.NodeType() != "New" {
		t.Errorf(errNodeType, n.NodeType())
	}
	if n.GetPos().Line != 2 || n.GetPos().Column != 2 {
		t.Errorf(errGetPos, n.GetPos())
	}
	n.SetPos(Position{Line: 3, Column: 3})
	if n.GetPos().Line != 3 || n.GetPos().Column != 3 {
		t.Errorf(errSetPos, n.GetPos())
	}
	if n.String() == "" {
		t.Error(errStringEmpty)
	}
	if n.TokenLiteral() != "new" {
		t.Errorf(errTokenLiteral, n.TokenLiteral())
	}
}

func TestMethodCallNode(t *testing.T) {
	obj := &VariableNode{Name: "obj", Pos: Position{Line: 1, Column: 1}}
	args := []Node{&IntegerLiteral{Value: 42, Pos: Position{Line: 1, Column: 2}}}
	m := &MethodCallNode{Object: obj, Method: "doSomething", Args: args, Pos: Position{Line: 2, Column: 2}}
	if m.NodeType() != "MethodCall" {
		t.Errorf(errNodeType, m.NodeType())
	}
	if m.GetPos().Line != 2 || m.GetPos().Column != 2 {
		t.Errorf(errGetPos, m.GetPos())
	}
	m.SetPos(Position{Line: 3, Column: 3})
	if m.GetPos().Line != 3 || m.GetPos().Column != 3 {
		t.Errorf(errSetPos, m.GetPos())
	}
	if m.String() == "" {
		t.Error(errStringEmpty)
	}
	if m.TokenLiteral() != "doSomething" {
		t.Errorf(errTokenLiteral, m.TokenLiteral())
	}
}

func TestTraitNode(t *testing.T) {
	name := &Identifier{Name: "MyTrait", Pos: Position{Line: 1, Column: 1}}
	body := []Node{&StringLiteral{Value: "body", Pos: Position{Line: 2, Column: 2}}}
	trait := &TraitNode{Name: name, Body: body, Pos: Position{Line: 3, Column: 3}}
	if trait.NodeType() != "Trait" {
		t.Errorf(errNodeType, trait.NodeType())
	}
	if trait.GetPos().Line != 3 || trait.GetPos().Column != 3 {
		t.Errorf(errGetPos, trait.GetPos())
	}
	trait.SetPos(Position{Line: 4, Column: 4})
	if trait.GetPos().Line != 4 || trait.GetPos().Column != 4 {
		t.Errorf(errSetPos, trait.GetPos())
	}
	if trait.String() == "" {
		t.Error(errStringEmpty)
	}
	if trait.TokenLiteral() != "trait" {
		t.Errorf(errTokenLiteral, trait.TokenLiteral())
	}
}
