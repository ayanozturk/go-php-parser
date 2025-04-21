package ast

import "testing"

func TestBlockNodeMethods(t *testing.T) {
	block := &BlockNode{Statements: []Node{}, Pos: Position{Line: 1, Column: 2}}
	if block.NodeType() != "Block" {
		t.Errorf("NodeType: got %q", block.NodeType())
	}
	if block.GetPos().Line != 1 || block.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", block.GetPos())
	}
	block.SetPos(Position{Line: 3, Column: 4})
	if block.GetPos().Line != 3 || block.GetPos().Column != 4 {
		t.Errorf("SetPos: got %+v", block.GetPos())
	}
	if block.String() != "Block @ 3:4" {
		t.Errorf("String: got %q", block.String())
	}
	if block.TokenLiteral() != "{" {
		t.Errorf("TokenLiteral: got %q", block.TokenLiteral())
	}
}

func TestIdentifierMethods(t *testing.T) {
	id := &Identifier{Name: "foo", Pos: Position{Line: 1, Column: 2}}
	if id.NodeType() != "Identifier" {
		t.Errorf("NodeType: got %q", id.NodeType())
	}
	if id.GetPos().Line != 1 || id.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", id.GetPos())
	}
	id.SetPos(Position{Line: 2, Column: 3})
	if id.GetPos().Line != 2 || id.GetPos().Column != 3 {
		t.Errorf("SetPos: got %+v", id.GetPos())
	}
	if id.String() != "Identifier(foo) @ 2:3" {
		t.Errorf("String: got %q", id.String())
	}
	if id.TokenLiteral() != "foo" {
		t.Errorf("TokenLiteral: got %q", id.TokenLiteral())
	}
}

func TestHeredocNodeMethods(t *testing.T) {
	parts := []Node{&StringLiteral{Value: "abc", Pos: Position{Line: 2, Column: 3}}}
	h := &HeredocNode{Identifier: "EOT", Parts: parts, Pos: Position{Line: 1, Column: 2}}
	if h.NodeType() != "Heredoc" {
		t.Errorf("NodeType: got %q", h.NodeType())
	}
	if h.GetPos().Line != 1 || h.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", h.GetPos())
	}
	h.SetPos(Position{Line: 3, Column: 4})
	if h.GetPos().Line != 3 || h.GetPos().Column != 4 {
		t.Errorf("SetPos: got %+v", h.GetPos())
	}
	if h.String() != "<<<'EOT' @ 3:4" {
		t.Errorf("String: got %q", h.String())
	}
	if h.TokenLiteral() != "EOT" {
		t.Errorf("TokenLiteral: got %q", h.TokenLiteral())
	}
}

func TestYieldNodeMethods(t *testing.T) {
	y := &YieldNode{From: true, Pos: Position{Line: 1, Column: 2}}
	if y.NodeType() != "Yield" {
		t.Errorf("NodeType: got %q", y.NodeType())
	}
	if y.GetPos().Line != 1 || y.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", y.GetPos())
	}
	y.SetPos(Position{Line: 3, Column: 4})
	if y.GetPos().Line != 3 || y.GetPos().Column != 4 {
		t.Errorf("SetPos: got %+v", y.GetPos())
	}
	if y.String() != "yield from @ 3:4" {
		t.Errorf("String: got %q", y.String())
	}
	if y.TokenLiteral() != "yield" {
		t.Errorf("TokenLiteral: got %q", y.TokenLiteral())
	}
}

func TestTypeCastNodeMethods(t *testing.T) {
	// Avoid shadowing t (*testing.T)
	tc := &TypeCastNode{Type: "int", Pos: Position{Line: 1, Column: 2}}
	if tc.NodeType() != "TypeCast" {
		t.Errorf("NodeType: got %q", tc.NodeType())
	}
	if tc.GetPos().Line != 1 || tc.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", tc.GetPos())
	}
	tc.SetPos(Position{Line: 3, Column: 4})
	if tc.GetPos().Line != 3 || tc.GetPos().Column != 4 {
		t.Errorf("SetPos: got %+v", tc.GetPos())
	}
	if tc.String() != "(int) @ 3:4" {
		t.Errorf("String: got %q", tc.String())
	}
	if tc.TokenLiteral() != "int" {
		t.Errorf("TokenLiteral: got %q", tc.TokenLiteral())
	}
}
