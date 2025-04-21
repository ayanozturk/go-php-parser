package ast

import "testing"

func TestCommentNodeMethods(t *testing.T) {
	c := &CommentNode{
		Value: "// hello world",
		Pos:   Position{Line: 7, Column: 2},
	}
	if c.NodeType() != "Comment" {
		t.Errorf("NodeType: got %q", c.NodeType())
	}
	if c.GetPos().Line != 7 || c.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", c.GetPos())
	}
	c.SetPos(Position{Line: 8, Column: 3})
	if c.GetPos().Line != 8 || c.GetPos().Column != 3 {
		t.Errorf("SetPos: got %+v", c.GetPos())
	}
	str := c.String()
	if str == "" || str == "Comment() @ 8:3" {
		t.Errorf("String: got %q", str)
	}
	if c.TokenLiteral() != "// hello world" {
		t.Errorf("TokenLiteral: got %q", c.TokenLiteral())
	}
}
