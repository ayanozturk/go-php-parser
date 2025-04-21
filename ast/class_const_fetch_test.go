package ast

import "testing"

func TestClassConstFetchNodeMethods(t *testing.T) {
	n := &ClassConstFetchNode{
		Class: "MyClass",
		Const: "MY_CONST",
		Pos:   Position{Line: 3, Column: 4},
	}
	if n.NodeType() != "ClassConstFetchNode" {
		t.Errorf("NodeType: got %q", n.NodeType())
	}
	if n.GetPos().Line != 3 || n.GetPos().Column != 4 {
		t.Errorf("GetPos: got %+v", n.GetPos())
	}
	n.SetPos(Position{Line: 5, Column: 6})
	if n.GetPos().Line != 5 || n.GetPos().Column != 6 {
		t.Errorf("SetPos: got %+v", n.GetPos())
	}
	str := n.String()
	if str != "MyClass::MY_CONST" {
		t.Errorf("String: got %q", str)
	}
	if n.TokenLiteral() != "MyClass::MY_CONST" {
		t.Errorf("TokenLiteral: got %q", n.TokenLiteral())
	}
}
