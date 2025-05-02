package ast

import (
	"testing"
)

const (
	errNodeType     = "NodeType: got %q"
	errGetPos       = "GetPos: got %+v"
	errSetPos       = "SetPos: got %+v"
	errStringEmpty  = "String should not be empty"
	errTokenLiteral = "TokenLiteral: got %q"
	errGetValue     = "GetValue: got %v"
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
		t.Errorf(errNodeType, id.NodeType())
	}
	if id.GetPos().Line != 1 || id.GetPos().Column != 2 {
		t.Errorf(errGetPos, id.GetPos())
	}
	id.SetPos(Position{Line: 3, Column: 4})
	if id.GetPos().Line != 3 || id.GetPos().Column != 4 {
		t.Errorf(errSetPos, id.GetPos())
	}
	if id.String() == "" {
		t.Error(errStringEmpty)
	}
	if id.TokenLiteral() != "foo" {
		t.Errorf(errTokenLiteral, id.TokenLiteral())
	}
}

func TestVariableNode(t *testing.T) {
	v := &VariableNode{Name: "bar", Pos: Position{Line: 2, Column: 3}}
	if v.NodeType() != "Variable" {
		t.Errorf(errNodeType, v.NodeType())
	}
	if v.GetPos().Line != 2 || v.GetPos().Column != 3 {
		t.Errorf(errGetPos, v.GetPos())
	}
	v.SetPos(Position{Line: 4, Column: 5})
	if v.GetPos().Line != 4 || v.GetPos().Column != 5 {
		t.Errorf(errSetPos, v.GetPos())
	}
	if v.String() == "" {
		t.Error(errStringEmpty)
	}
	if v.TokenLiteral() != "bar" {
		t.Errorf(errTokenLiteral, v.TokenLiteral())
	}
}

func TestStringLiteralNode(t *testing.T) {
	s := &StringLiteral{Value: "baz", Pos: Position{Line: 3, Column: 4}}
	if s.NodeType() != "StringLiteral" {
		t.Errorf(errNodeType, s.NodeType())
	}
	if s.GetPos().Line != 3 || s.GetPos().Column != 4 {
		t.Errorf(errGetPos, s.GetPos())
	}
	s.SetPos(Position{Line: 5, Column: 6})
	if s.GetPos().Line != 5 || s.GetPos().Column != 6 {
		t.Errorf(errSetPos, s.GetPos())
	}
	if s.String() == "" {
		t.Error(errStringEmpty)
	}
	if s.TokenLiteral() != "baz" {
		t.Errorf(errTokenLiteral, s.TokenLiteral())
	}
	if val, ok := s.GetValue().(string); !ok || val != "baz" {
		t.Errorf(errGetValue, s.GetValue())
	}
}

func TestIntegerLiteralNode(t *testing.T) {
	i := &IntegerLiteral{Value: 42, Pos: Position{Line: 6, Column: 7}}
	if i.NodeType() != "IntegerLiteral" {
		t.Errorf(errNodeType, i.NodeType())
	}
	if i.GetPos().Line != 6 || i.GetPos().Column != 7 {
		t.Errorf(errGetPos, i.GetPos())
	}
	i.SetPos(Position{Line: 8, Column: 9})
	if i.GetPos().Line != 8 || i.GetPos().Column != 9 {
		t.Errorf(errSetPos, i.GetPos())
	}
	if i.String() == "" {
		t.Error(errStringEmpty)
	}
	if i.TokenLiteral() != "42" {
		t.Errorf(errTokenLiteral, i.TokenLiteral())
	}
	if val, ok := i.GetValue().(int64); !ok || val != 42 {
		t.Errorf(errGetValue, i.GetValue())
	}
}

func TestFloatLiteralNode(t *testing.T) {
	f := &FloatLiteral{Value: 3.14, Pos: Position{Line: 10, Column: 11}}
	if f.NodeType() != "FloatLiteral" {
		t.Errorf(errNodeType, f.NodeType())
	}
	if f.GetPos().Line != 10 || f.GetPos().Column != 11 {
		t.Errorf(errGetPos, f.GetPos())
	}
	f.SetPos(Position{Line: 12, Column: 13})
	if f.GetPos().Line != 12 || f.GetPos().Column != 13 {
		t.Errorf(errSetPos, f.GetPos())
	}
	if f.String() == "" {
		t.Error(errStringEmpty)
	}
	if f.TokenLiteral() != "3.14" {
		t.Errorf(errTokenLiteral, f.TokenLiteral())
	}
	if val, ok := f.GetValue().(float64); !ok || val != 3.14 {
		t.Errorf(errGetValue, f.GetValue())
	}
}

func TestBlockNode(t *testing.T) {
	stmt := &Identifier{Name: "stmt", Pos: Position{Line: 1, Column: 1}}
	block := &BlockNode{Statements: []Node{stmt}, Pos: Position{Line: 2, Column: 2}}
	if block.NodeType() != "Block" {
		t.Errorf(errNodeType, block.NodeType())
	}
	if block.GetPos().Line != 2 || block.GetPos().Column != 2 {
		t.Errorf(errGetPos, block.GetPos())
	}
	block.SetPos(Position{Line: 3, Column: 3})
	if block.GetPos().Line != 3 || block.GetPos().Column != 3 {
		t.Errorf(errSetPos, block.GetPos())
	}
	if block.String() == "" {
		t.Error(errStringEmpty)
	}
	if block.TokenLiteral() != "{" {
		t.Errorf(errTokenLiteral, block.TokenLiteral())
	}
	if len(block.Statements) != 1 || block.Statements[0] != stmt {
		t.Errorf("Statements: got %+v", block.Statements)
	}
}

func TestTypeCastNode(t *testing.T) {
	typeCast := &TypeCastNode{Type: "int", Expr: &Identifier{Name: "foo"}, Pos: Position{Line: 1, Column: 1}}
	if typeCast.NodeType() != "TypeCast" {
		t.Errorf(errNodeType, typeCast.NodeType())
	}
	if typeCast.GetPos().Line != 1 {
		t.Errorf(errGetPos, typeCast.GetPos())
	}
	typeCast.SetPos(Position{Line: 2, Column: 2})
	if typeCast.GetPos().Line != 2 {
		t.Errorf(errSetPos, typeCast.GetPos())
	}
	if typeCast.String() == "" {
		t.Error(errStringEmpty)
	}
	if typeCast.TokenLiteral() != "int" {
		t.Errorf(errTokenLiteral, typeCast.TokenLiteral())
	}
}

func TestYieldNode(t *testing.T) {
	yield := &YieldNode{Key: nil, Value: &Identifier{Name: "foo"}, From: true, Pos: Position{Line: 1, Column: 1}}
	if yield.NodeType() != "Yield" {
		t.Errorf(errNodeType, yield.NodeType())
	}
	if yield.GetPos().Line != 1 {
		t.Errorf(errGetPos, yield.GetPos())
	}
	yield.SetPos(Position{Line: 2, Column: 2})
	if yield.GetPos().Line != 2 {
		t.Errorf(errSetPos, yield.GetPos())
	}
	if yield.String() == "" {
		t.Error(errStringEmpty)
	}
	if yield.TokenLiteral() != "yield" {
		t.Errorf(errTokenLiteral, yield.TokenLiteral())
	}
}

func TestHeredocNode(t *testing.T) {
	heredoc := &HeredocNode{Identifier: "EOT", Parts: []Node{&Identifier{Name: "foo"}}, Pos: Position{Line: 1, Column: 1}}
	if heredoc.NodeType() != "Heredoc" {
		t.Errorf(errNodeType, heredoc.NodeType())
	}
	if heredoc.GetPos().Line != 1 {
		t.Errorf(errGetPos, heredoc.GetPos())
	}
	heredoc.SetPos(Position{Line: 2, Column: 2})
	if heredoc.GetPos().Line != 2 {
		t.Errorf(errSetPos, heredoc.GetPos())
	}
	if heredoc.String() == "" {
		t.Error(errStringEmpty)
	}
	if heredoc.TokenLiteral() != "EOT" {
		t.Errorf(errTokenLiteral, heredoc.TokenLiteral())
	}
}

func TestTernaryExpr(t *testing.T) {
	ternary := &TernaryExpr{Condition: &Identifier{Name: "cond"}, IfTrue: &Identifier{Name: "yes"}, IfFalse: &Identifier{Name: "no"}, Pos: Position{Line: 1, Column: 1}}
	if ternary.NodeType() != "TernaryExpr" {
		t.Errorf(errNodeType, ternary.NodeType())
	}
	if ternary.GetPos().Line != 1 {
		t.Errorf(errGetPos, ternary.GetPos())
	}
	ternary.SetPos(Position{Line: 2, Column: 2})
	if ternary.GetPos().Line != 2 {
		t.Errorf(errSetPos, ternary.GetPos())
	}
	if ternary.String() == "" {
		t.Error(errStringEmpty)
	}
	if ternary.TokenLiteral() != "?" {
		t.Errorf(errTokenLiteral, ternary.TokenLiteral())
	}
}

func TestPropertyFetchNode(t *testing.T) {
	obj := &Identifier{Name: "obj"}
	prop := "prop"
	pf := &PropertyFetchNode{Object: obj, Property: prop, Pos: Position{Line: 1, Column: 1}}
	if pf.NodeType() != "PropertyFetch" {
		t.Errorf(errNodeType, pf.NodeType())
	}
	if pf.GetPos().Line != 1 {
		t.Errorf(errGetPos, pf.GetPos())
	}
	pf.SetPos(Position{Line: 2, Column: 2})
	if pf.GetPos().Line != 2 {
		t.Errorf(errSetPos, pf.GetPos())
	}
	if pf.String() == "" {
		t.Error(errStringEmpty)
	}
	if pf.TokenLiteral() != prop {
		t.Errorf(errTokenLiteral, pf.TokenLiteral())
	}
}

func TestForeachNode(t *testing.T) {
	expr := &Identifier{Name: "arr"}
	key := &Identifier{Name: "k"}
	val := &Identifier{Name: "v"}
	body := []Node{&Identifier{Name: "stmt"}}
	foreach := &ForeachNode{Expr: expr, KeyVar: key, ValueVar: val, ByRef: false, Body: body, Pos: Position{Line: 1, Column: 1}}
	if foreach.NodeType() != "Foreach" {
		t.Errorf(errNodeType, foreach.NodeType())
	}
	if foreach.GetPos().Line != 1 {
		t.Errorf(errGetPos, foreach.GetPos())
	}
	foreach.SetPos(Position{Line: 2, Column: 2})
	if foreach.GetPos().Line != 2 {
		t.Errorf(errSetPos, foreach.GetPos())
	}
	if foreach.String() == "" {
		t.Error(errStringEmpty)
	}
	if foreach.TokenLiteral() != "foreach" {
		t.Errorf(errTokenLiteral, foreach.TokenLiteral())
	}
}

func TestThrowNode(t *testing.T) {
	throw := &ThrowNode{Expr: &Identifier{Name: "ex"}, Pos: Position{Line: 1, Column: 1}}
	if throw.NodeType() != "Throw" {
		t.Errorf(errNodeType, throw.NodeType())
	}
	if throw.GetPos().Line != 1 {
		t.Errorf(errGetPos, throw.GetPos())
	}
	throw.SetPos(Position{Line: 2, Column: 2})
	if throw.GetPos().Line != 2 {
		t.Errorf(errSetPos, throw.GetPos())
	}
	if throw.String() == "" {
		t.Error(errStringEmpty)
	}
	if throw.TokenLiteral() != "throw" {
		t.Errorf(errTokenLiteral, throw.TokenLiteral())
	}
}
