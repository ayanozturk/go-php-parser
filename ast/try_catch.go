package ast

import "fmt"

type TryNode struct {
	Body    []Node
	Catches []*CatchNode
	Finally []Node
	Pos     Position
	EndPos  Position
}

func (t *TryNode) NodeType() string       { return "Try" }
func (t *TryNode) GetPos() Position       { return t.Pos }
func (t *TryNode) SetPos(pos Position)    { t.Pos = pos }
func (t *TryNode) GetEndPos() Position    { return t.EndPos }
func (t *TryNode) SetEndPos(pos Position) { t.EndPos = pos }
func (t *TryNode) String() string {
	return fmt.Sprintf("Try @ %d:%d", t.Pos.Line, t.Pos.Column)
}
func (t *TryNode) TokenLiteral() string { return "try" }

type CatchNode struct {
	Types    []string
	Variable string
	Body     []Node
	Pos      Position
	EndPos   Position
}

func (c *CatchNode) NodeType() string       { return "Catch" }
func (c *CatchNode) GetPos() Position       { return c.Pos }
func (c *CatchNode) SetPos(pos Position)    { c.Pos = pos }
func (c *CatchNode) GetEndPos() Position    { return c.EndPos }
func (c *CatchNode) SetEndPos(pos Position) { c.EndPos = pos }
func (c *CatchNode) String() string {
	return fmt.Sprintf("Catch @ %d:%d", c.Pos.Line, c.Pos.Column)
}
func (c *CatchNode) TokenLiteral() string { return "catch" }
