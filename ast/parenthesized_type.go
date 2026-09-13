package ast

import "fmt"

// ParenthesizedTypeNode represents (T) used in DNF types.
type ParenthesizedTypeNode struct {
	Inner  Node
	Pos    Position
	EndPos Position
}

func (n *ParenthesizedTypeNode) NodeType() string       { return "ParenthesizedType" }
func (n *ParenthesizedTypeNode) GetPos() Position       { return n.Pos }
func (n *ParenthesizedTypeNode) SetPos(pos Position)    { n.Pos = pos }
func (n *ParenthesizedTypeNode) GetEndPos() Position    { return n.EndPos }
func (n *ParenthesizedTypeNode) SetEndPos(pos Position) { n.EndPos = pos }
func (n *ParenthesizedTypeNode) TokenLiteral() string {
	return "(" + TypeText(n.Inner) + ")"
}
func (n *ParenthesizedTypeNode) String() string {
	return fmt.Sprintf("ParenthesizedType(%s) @ %d:%d", TypeText(n.Inner), n.Pos.Line, n.Pos.Column)
}
