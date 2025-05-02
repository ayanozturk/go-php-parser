package ast

import (
	"reflect"
	"testing"
)

func TestDeclareNode_Basic(t *testing.T) {
	directives := map[string]Node{"strict_types": &IntegerLiteral{Value: 1}}
	pos := Position{Line: 1, Column: 2}
	body := &BlockNode{Statements: []Node{}, Pos: pos}
	n := &DeclareNode{Directives: directives, Pos: pos, Body: body}

	if n.NodeType() != "Declare" {
		t.Errorf("NodeType() = %s; want Declare", n.NodeType())
	}
	if n.GetPos() != pos {
		t.Errorf("GetPos() = %+v; want %+v", n.GetPos(), pos)
	}
	n.SetPos(Position{Line: 2, Column: 3})
	if n.GetPos() != (Position{Line: 2, Column: 3}) {
		t.Errorf("SetPos() failed, got %+v", n.GetPos())
	}
	if n.TokenLiteral() != "declare" {
		t.Errorf("TokenLiteral() = %s; want declare", n.TokenLiteral())
	}
	if n.String() == "" {
		t.Error("String() should not be empty")
	}
	if !reflect.DeepEqual(n.Directives, directives) {
		t.Errorf("Directives = %+v; want %+v", n.Directives, directives)
	}
	if n.Body != body {
		t.Errorf("Body = %+v; want %+v", n.Body, body)
	}
}

func TestDeclareDirective_Basic(t *testing.T) {
	val := &IntegerLiteral{Value: 1}
	pos := Position{Line: 4, Column: 5}
	d := &DeclareDirective{Name: "strict_types", Value: val, Pos: pos}

	if d.NodeType() != "DeclareDirective" {
		t.Errorf("NodeType() = %s; want DeclareDirective", d.NodeType())
	}
	if d.GetPos() != pos {
		t.Errorf("GetPos() = %+v; want %+v", d.GetPos(), pos)
	}
	d.SetPos(Position{Line: 6, Column: 7})
	if d.GetPos() != (Position{Line: 6, Column: 7}) {
		t.Errorf("SetPos() failed, got %+v", d.GetPos())
	}
	if d.TokenLiteral() != "strict_types" {
		t.Errorf("TokenLiteral() = %s; want strict_types", d.TokenLiteral())
	}
	if d.String() == "" {
		t.Error("String() should not be empty")
	}
	if d.Value != val {
		t.Errorf("Value = %+v; want %+v", d.Value, val)
	}
	if d.Name != "strict_types" {
		t.Errorf("Name = %s; want strict_types", d.Name)
	}
}
