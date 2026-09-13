package ast

import "fmt"

// NullableTypeNode represents a ?T native type.
type NullableTypeNode struct {
	Inner  Node
	Pos    Position
	EndPos Position
}

func (n *NullableTypeNode) NodeType() string       { return "NullableType" }
func (n *NullableTypeNode) GetPos() Position       { return n.Pos }
func (n *NullableTypeNode) SetPos(pos Position)    { n.Pos = pos }
func (n *NullableTypeNode) GetEndPos() Position    { return n.EndPos }
func (n *NullableTypeNode) SetEndPos(pos Position) { n.EndPos = pos }
func (n *NullableTypeNode) TokenLiteral() string {
	return "?" + TypeText(n.Inner)
}
func (n *NullableTypeNode) String() string {
	return fmt.Sprintf("NullableType(%s) @ %d:%d", TypeText(n.Inner), n.Pos.Line, n.Pos.Column)
}
