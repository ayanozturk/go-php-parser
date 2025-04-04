package ast

import "fmt"

// CommentNode represents a comment in the code
type CommentNode struct {
	Value string
	Pos   Position
}

func (c *CommentNode) NodeType() string    { return "Comment" }
func (c *CommentNode) GetPos() Position    { return c.Pos }
func (c *CommentNode) SetPos(pos Position) { c.Pos = pos }
func (c *CommentNode) String() string {
	return fmt.Sprintf("Comment(%s) @ %d:%d", c.Value, c.Pos.Line, c.Pos.Column)
}
func (c *CommentNode) TokenLiteral() string {
	return c.Value
}
