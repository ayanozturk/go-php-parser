package ast

import "fmt"

// ConstantNode represents a PHP constant declaration
// e.g. const FOO = 123;
type ConstantNode struct {
	Name  string
	Value Node
	Pos   Position
}

func (c *ConstantNode) NodeType() string    { return "Constant" }
func (c *ConstantNode) GetPos() Position    { return c.Pos }
func (c *ConstantNode) SetPos(pos Position) { c.Pos = pos }
func (c *ConstantNode) String() string {
	return fmt.Sprintf("Constant(%s = %s) @ %d:%d", c.Name, c.Value.TokenLiteral(), c.Pos.Line, c.Pos.Column)
}
func (c *ConstantNode) TokenLiteral() string {
	return c.Name
}
