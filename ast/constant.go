package ast

import "fmt"

// ConstantNode represents a PHP constant declaration
// e.g. public const FOO: int = 123;
type ConstantNode struct {
	Name       string
	Type       string // e.g. "int", "string", etc.
	Visibility string // "public", "protected", "private", or ""
	Value      Node
	Pos        Position
}

func (c *ConstantNode) NodeType() string    { return "Constant" }
func (c *ConstantNode) GetPos() Position    { return c.Pos }
func (c *ConstantNode) SetPos(pos Position) { c.Pos = pos }
func (c *ConstantNode) String() string {
	return fmt.Sprintf("Constant(%s %s: %s = %s) @ %d:%d", c.Visibility, c.Name, c.Type, c.Value.TokenLiteral(), c.Pos.Line, c.Pos.Column)
}
func (c *ConstantNode) TokenLiteral() string {
	return c.Name
}
