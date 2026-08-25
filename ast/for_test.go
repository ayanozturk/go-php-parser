package ast

import "testing"

func TestForNodeMethodsAndSpan(t *testing.T) {
	loop := &ForNode{
		Init:       []Node{&IntegerLiteral{Value: 0}},
		Conditions: []Node{&BooleanLiteral{Value: true}},
		Updates:    []Node{&IntegerLiteral{Value: 1}},
		Body:       []Node{&ExpressionStmt{Expr: &IntegerLiteral{Value: 2}}},
		Pos:        Position{Line: 2, Column: 3, Offset: 10},
		EndPos:     Position{Line: 4, Column: 2, Offset: 42},
	}

	if got := loop.NodeType(); got != "For" {
		t.Fatalf("NodeType() = %q, want For", got)
	}
	if got := loop.TokenLiteral(); got != "for" {
		t.Fatalf("TokenLiteral() = %q, want for", got)
	}
	if got := loop.String(); got != "For(init: 1, conditions: 1, updates: 1) @ 2:3" {
		t.Fatalf("String() = %q", got)
	}
	if got := loop.GetPos(); got != loop.Pos {
		t.Fatalf("GetPos() = %+v, want %+v", got, loop.Pos)
	}
	if got := loop.GetEndPos(); got != loop.EndPos {
		t.Fatalf("GetEndPos() = %+v, want %+v", got, loop.EndPos)
	}

	newPos := Position{Line: 6, Column: 7, Offset: 60}
	loop.SetPos(newPos)
	if got := loop.GetPos(); got != newPos {
		t.Fatalf("SetPos/GetPos = %+v, want %+v", got, newPos)
	}
	newEnd := Position{Line: 8, Column: 1, Offset: 90}
	loop.SetEndPos(newEnd)
	if got := loop.GetEndPos(); got != newEnd {
		t.Fatalf("SetEndPos/GetEndPos = %+v, want %+v", got, newEnd)
	}
}
