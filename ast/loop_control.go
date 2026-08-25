package ast

import "fmt"

// BreakNode transfers control out of the requested number of enclosing loop
// or switch scopes. A nil Level represents PHP's implicit level one.
type BreakNode struct {
	Level  Node
	Pos    Position
	EndPos Position
}

func (b *BreakNode) NodeType() string       { return "Break" }
func (b *BreakNode) GetPos() Position       { return b.Pos }
func (b *BreakNode) SetPos(pos Position)    { b.Pos = pos }
func (b *BreakNode) GetEndPos() Position    { return b.EndPos }
func (b *BreakNode) SetEndPos(pos Position) { b.EndPos = pos }
func (b *BreakNode) String() string {
	if b.Level == nil {
		return fmt.Sprintf("Break @ %d:%d", b.Pos.Line, b.Pos.Column)
	}
	return fmt.Sprintf("Break(%s) @ %d:%d", b.Level.TokenLiteral(), b.Pos.Line, b.Pos.Column)
}
func (b *BreakNode) TokenLiteral() string { return "break" }

// ContinueNode transfers control to the next iteration of the requested
// enclosing loop scope. A nil Level represents PHP's implicit level one.
type ContinueNode struct {
	Level  Node
	Pos    Position
	EndPos Position
}

func (c *ContinueNode) NodeType() string       { return "Continue" }
func (c *ContinueNode) GetPos() Position       { return c.Pos }
func (c *ContinueNode) SetPos(pos Position)    { c.Pos = pos }
func (c *ContinueNode) GetEndPos() Position    { return c.EndPos }
func (c *ContinueNode) SetEndPos(pos Position) { c.EndPos = pos }
func (c *ContinueNode) String() string {
	if c.Level == nil {
		return fmt.Sprintf("Continue @ %d:%d", c.Pos.Line, c.Pos.Column)
	}
	return fmt.Sprintf("Continue(%s) @ %d:%d", c.Level.TokenLiteral(), c.Pos.Line, c.Pos.Column)
}
func (c *ContinueNode) TokenLiteral() string { return "continue" }
