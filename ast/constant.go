package ast

import "fmt"

// ConstantNode represents a PHP constant declaration
// e.g. public const FOO: int = 123;
type ConstantNode struct {
	Name       string
	Type       Node // typed class const type hint, nil if untyped
	Modifiers  ModifierList
	Value      Node
	Attributes []Node
	PHPDoc     *PHPDocNode // Associated PHPDoc comment
	Pos        Position
	EndPos     Position
}

func (c *ConstantNode) NodeType() string       { return "Constant" }
func (c *ConstantNode) GetPos() Position       { return c.Pos }
func (c *ConstantNode) SetPos(pos Position)    { c.Pos = pos }
func (c *ConstantNode) GetEndPos() Position    { return c.EndPos }
func (c *ConstantNode) SetEndPos(pos Position) { c.EndPos = pos }
func (c *ConstantNode) String() string {
	return fmt.Sprintf("Constant(%s %s: %s = %s) @ %d:%d", c.Modifiers.Visibility(), c.Name, TypeText(c.Type), c.Value.TokenLiteral(), c.Pos.Line, c.Pos.Column)
}
func (c *ConstantNode) TokenLiteral() string {
	return c.Name
}
