package ast

import "testing"

func TestInterfaceNodeMethods(t *testing.T) {
	method := &InterfaceMethodNode{
		Name:       "doSomething",
		Visibility: "public",
		ReturnType: &UnionTypeNode{Types: []string{"int", "string"}, Pos: Position{Line: 5, Column: 6}},
		Params:     []Node{&ParamNode{Name: "x", TypeHint: "int", Pos: Position{Line: 7, Column: 8}}},
		Pos:        Position{Line: 4, Column: 4},
	}
	iface := &InterfaceNode{
		Name:    "MyInterface",
		Extends: []string{"BaseInterface"},
		Members: []Node{method},
		Pos:     Position{Line: 1, Column: 1},
	}
	if iface.NodeType() != "Interface" {
		t.Errorf("NodeType: got %q", iface.NodeType())
	}
	if iface.GetPos().Line != 1 || iface.GetPos().Column != 1 {
		t.Errorf("GetPos: got %+v", iface.GetPos())
	}
	iface.SetPos(Position{Line: 2, Column: 2})
	if iface.GetPos().Line != 2 || iface.GetPos().Column != 2 {
		t.Errorf("SetPos: got %+v", iface.GetPos())
	}
	str := iface.String()
	if str == "" {
		t.Errorf("String: got %q", str)
	}
	if iface.TokenLiteral() != "interface" {
		t.Errorf("TokenLiteral: got %q", iface.TokenLiteral())
	}
	if len(iface.Members) != 1 || iface.Members[0] != method {
		t.Errorf("Members: got %+v", iface.Members)
	}
}

func TestInterfaceMethodNodeMethods(t *testing.T) {
	param := &ParamNode{Name: "y", TypeHint: "string", Pos: Position{Line: 8, Column: 9}}
	ret := &UnionTypeNode{Types: []string{"float"}, Pos: Position{Line: 10, Column: 11}}
	method := &InterfaceMethodNode{
		Name:       "bar",
		Visibility: "protected",
		ReturnType: ret,
		Params:     []Node{param},
		Pos:        Position{Line: 7, Column: 7},
	}
	if method.NodeType() != "InterfaceMethod" {
		t.Errorf("NodeType: got %q", method.NodeType())
	}
	if method.GetPos().Line != 7 || method.GetPos().Column != 7 {
		t.Errorf("GetPos: got %+v", method.GetPos())
	}
	method.SetPos(Position{Line: 9, Column: 10})
	if method.GetPos().Line != 9 || method.GetPos().Column != 10 {
		t.Errorf("SetPos: got %+v", method.GetPos())
	}
	str := method.String()
	if str == "" {
		t.Errorf("String: got %q", str)
	}
	if method.TokenLiteral() != "function" {
		t.Errorf("TokenLiteral: got %q", method.TokenLiteral())
	}
	if len(method.Params) != 1 || method.Params[0] != param {
		t.Errorf("Params: got %+v", method.Params)
	}
}
