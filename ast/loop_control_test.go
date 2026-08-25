package ast

import "testing"

func TestBreakNodeMethodsImplicitLevelAndSpan(t *testing.T) {
	node := &BreakNode{
		Pos:    Position{Line: 2, Column: 5, Offset: 12},
		EndPos: Position{Line: 2, Column: 11, Offset: 18},
	}

	if node.Level != nil {
		t.Fatalf("implicit break level = %T, want nil", node.Level)
	}
	if got := node.NodeType(); got != "Break" {
		t.Fatalf("NodeType() = %q, want Break", got)
	}
	if got := node.TokenLiteral(); got != "break" {
		t.Fatalf("TokenLiteral() = %q, want break", got)
	}
	if got := node.String(); got != "Break @ 2:5" {
		t.Fatalf("String() = %q, want %q", got, "Break @ 2:5")
	}
	assertLoopControlSpan(t, node, node.Pos, node.EndPos)
}

func TestContinueNodeMethodsNumericLevelAndSpan(t *testing.T) {
	node := &ContinueNode{
		Level:  &IntegerLiteral{Value: 2},
		Pos:    Position{Line: 4, Column: 7, Offset: 30},
		EndPos: Position{Line: 4, Column: 18, Offset: 41},
	}

	if got := node.NodeType(); got != "Continue" {
		t.Fatalf("NodeType() = %q, want Continue", got)
	}
	if got := node.TokenLiteral(); got != "continue" {
		t.Fatalf("TokenLiteral() = %q, want continue", got)
	}
	if got := node.String(); got != "Continue(2) @ 4:7" {
		t.Fatalf("String() = %q, want %q", got, "Continue(2) @ 4:7")
	}
	assertLoopControlSpan(t, node, node.Pos, node.EndPos)
}

func assertLoopControlSpan(t *testing.T, node Node, start, end Position) {
	t.Helper()
	if got := node.GetPos(); got != start {
		t.Fatalf("GetPos() = %+v, want %+v", got, start)
	}
	if got := node.GetEndPos(); got != end {
		t.Fatalf("GetEndPos() = %+v, want %+v", got, end)
	}

	newStart := Position{Line: 8, Column: 2, Offset: 80}
	newEnd := Position{Line: 8, Column: 13, Offset: 91}
	node.SetPos(newStart)
	node.SetEndPos(newEnd)
	if got := node.GetPos(); got != newStart {
		t.Fatalf("SetPos/GetPos = %+v, want %+v", got, newStart)
	}
	if got := node.GetEndPos(); got != newEnd {
		t.Fatalf("SetEndPos/GetEndPos = %+v, want %+v", got, newEnd)
	}
}
