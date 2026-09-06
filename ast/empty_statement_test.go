package ast

import "testing"

func TestEmptyStatementNodeMethodsAndSpan(t *testing.T) {
	node := &EmptyStatementNode{
		Pos:    Position{Line: 2, Column: 1, Offset: 6},
		EndPos: Position{Line: 2, Column: 2, Offset: 7},
	}
	if got := node.NodeType(); got != "EmptyStatement" {
		t.Fatalf("NodeType() = %q, want EmptyStatement", got)
	}
	if got := node.TokenLiteral(); got != ";" {
		t.Fatalf("TokenLiteral() = %q, want ;", got)
	}
	if got := node.String(); got != "EmptyStatement @ 2:1" {
		t.Fatalf("String() = %q, want %q", got, "EmptyStatement @ 2:1")
	}
	assertLoopControlSpan(t, node, node.Pos, node.EndPos)
}
