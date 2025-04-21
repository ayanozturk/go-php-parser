package ast

import (
	"testing"
)

func TestPositionStruct(t *testing.T) {
	pos := Position{Line: 1, Column: 2, Offset: 3}
	if pos.Line != 1 || pos.Column != 2 || pos.Offset != 3 {
		t.Errorf("unexpected Position values: %+v", pos)
	}
}

func TestIdentifierNode(t *testing.T) {
	id := &Identifier{Name: "foo", Pos: Position{Line: 1, Column: 2}}
	if id.NodeType() != "Identifier" {
		t.Errorf("NodeType: got %q", id.NodeType())
	}
	if id.GetPos().Line != 1 || id.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", id.GetPos())
	}
	id.SetPos(Position{Line: 3, Column: 4})
	if id.GetPos().Line != 3 || id.GetPos().Column != 4 {
		t.Errorf("SetPos: got %+v", id.GetPos())
	}
	if id.String() == "" {
		t.Error("String should not be empty")
	}
	if id.TokenLiteral() != "foo" {
		t.Errorf("TokenLiteral: got %q", id.TokenLiteral())
	}
}

func TestVariableNode(t *testing.T) {
	v := &VariableNode{Name: "bar", Pos: Position{Line: 2, Column: 3}}
	if v.NodeType() != "Variable" {
		t.Errorf("NodeType: got %q", v.NodeType())
	}
	if v.GetPos().Line != 2 || v.GetPos().Column != 3 {
		t.Errorf("GetPos: got %+v", v.GetPos())
	}
	v.SetPos(Position{Line: 4, Column: 5})
	if v.GetPos().Line != 4 || v.GetPos().Column != 5 {
		t.Errorf("SetPos: got %+v", v.GetPos())
	}
	if v.String() == "" {
		t.Error("String should not be empty")
	}
	if v.TokenLiteral() != "bar" {
		t.Errorf("TokenLiteral: got %q", v.TokenLiteral())
	}
}

func TestStringLiteralNode(t *testing.T) {
	s := &StringLiteral{Value: "baz", Pos: Position{Line: 3, Column: 4}}
	if s.NodeType() != "StringLiteral" {
		t.Errorf("NodeType: got %q", s.NodeType())
	}
	if s.GetPos().Line != 3 || s.GetPos().Column != 4 {
		t.Errorf("GetPos: got %+v", s.GetPos())
	}
	s.SetPos(Position{Line: 5, Column: 6})
	if s.GetPos().Line != 5 || s.GetPos().Column != 6 {
		t.Errorf("SetPos: got %+v", s.GetPos())
	}
	if s.String() == "" {
		t.Error("String should not be empty")
	}
	if s.TokenLiteral() != "baz" {
		t.Errorf("TokenLiteral: got %q", s.TokenLiteral())
	}
	if val, ok := s.GetValue().(string); !ok || val != "baz" {
		t.Errorf("GetValue: got %v", s.GetValue())
	}
}

func TestIntegerLiteralNode(t *testing.T) {
	i := &IntegerLiteral{Value: 42, Pos: Position{Line: 6, Column: 7}}
	if i.NodeType() != "IntegerLiteral" {
		t.Errorf("NodeType: got %q", i.NodeType())
	}
	if i.GetPos().Line != 6 || i.GetPos().Column != 7 {
		t.Errorf("GetPos: got %+v", i.GetPos())
	}
	i.SetPos(Position{Line: 8, Column: 9})
	if i.GetPos().Line != 8 || i.GetPos().Column != 9 {
		t.Errorf("SetPos: got %+v", i.GetPos())
	}
	if i.String() == "" {
		t.Error("String should not be empty")
	}
	if i.TokenLiteral() != "42" {
		t.Errorf("TokenLiteral: got %q", i.TokenLiteral())
	}
	if val, ok := i.GetValue().(int64); !ok || val != 42 {
		t.Errorf("GetValue: got %v", i.GetValue())
	}
}

func TestFloatLiteralNode(t *testing.T) {
	f := &FloatLiteral{Value: 3.14, Pos: Position{Line: 10, Column: 11}}
	if f.NodeType() != "FloatLiteral" {
		t.Errorf("NodeType: got %q", f.NodeType())
	}
	if f.GetPos().Line != 10 || f.GetPos().Column != 11 {
		t.Errorf("GetPos: got %+v", f.GetPos())
	}
	f.SetPos(Position{Line: 12, Column: 13})
	if f.GetPos().Line != 12 || f.GetPos().Column != 13 {
		t.Errorf("SetPos: got %+v", f.GetPos())
	}
	if f.String() == "" {
		t.Error("String should not be empty")
	}
	if f.TokenLiteral() != "3.14" {
		t.Errorf("TokenLiteral: got %q", f.TokenLiteral())
	}
	if val, ok := f.GetValue().(float64); !ok || val != 3.14 {
		t.Errorf("GetValue: got %v", f.GetValue())
	}
}

func TestBlockNode(t *testing.T) {
	stmt := &Identifier{Name: "stmt", Pos: Position{Line: 1, Column: 1}}
	block := &BlockNode{Statements: []Node{stmt}, Pos: Position{Line: 2, Column: 2}}
	if block.NodeType() != "Block" {
		t.Errorf("NodeType: got %q", block.NodeType())
	}
	if block.GetPos().Line != 2 || block.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", block.GetPos())
	}
	block.SetPos(Position{Line: 3, Column: 3})
	if block.GetPos().Line != 3 || block.GetPos().Column != 3 {
		t.Errorf("SetPos: got %+v", block.GetPos())
	}
	if block.String() == "" {
		t.Error("String should not be empty")
	}
	if block.TokenLiteral() != "{" {
		t.Errorf("TokenLiteral: got %q", block.TokenLiteral())
	}
	if len(block.Statements) != 1 || block.Statements[0] != stmt {
		t.Errorf("Statements: got %+v", block.Statements)
	}
}
