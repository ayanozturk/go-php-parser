package ast

import (
	"fmt"
	"strings"
)

// CallableParamNode is one callable(...) parameter: optional type + optional $name.
type CallableParamNode struct {
	TypeHint Node
	Name     string // without leading $, empty if unnamed
	Pos      Position
	EndPos   Position
}

func (n *CallableParamNode) NodeType() string       { return "CallableParam" }
func (n *CallableParamNode) GetPos() Position       { return n.Pos }
func (n *CallableParamNode) SetPos(pos Position)    { n.Pos = pos }
func (n *CallableParamNode) GetEndPos() Position    { return n.EndPos }
func (n *CallableParamNode) SetEndPos(pos Position) { n.EndPos = pos }
func (n *CallableParamNode) TokenLiteral() string {
	t := TypeText(n.TypeHint)
	if n.Name == "" {
		return t
	}
	if t == "" {
		return "$" + n.Name
	}
	return t + "$" + n.Name
}
func (n *CallableParamNode) String() string {
	return fmt.Sprintf("CallableParam(%s) @ %d:%d", n.TokenLiteral(), n.Pos.Line, n.Pos.Column)
}

// CallableTypeNode is bare callable or callable(...): ReturnType with nested child types.
type CallableTypeNode struct {
	Params       []*CallableParamNode
	HasSignature bool // true when '(' was present
	ReturnType   Node
	Pos          Position
	EndPos       Position
}

func (n *CallableTypeNode) NodeType() string       { return "CallableType" }
func (n *CallableTypeNode) GetPos() Position       { return n.Pos }
func (n *CallableTypeNode) SetPos(pos Position)    { n.Pos = pos }
func (n *CallableTypeNode) GetEndPos() Position    { return n.EndPos }
func (n *CallableTypeNode) SetEndPos(pos Position) { n.EndPos = pos }
func (n *CallableTypeNode) TokenLiteral() string {
	if !n.HasSignature {
		return "callable"
	}
	var b strings.Builder
	b.WriteString("callable(")
	for i, p := range n.Params {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(p.TokenLiteral())
	}
	b.WriteByte(')')
	if rt := TypeText(n.ReturnType); rt != "" {
		b.WriteByte(':')
		b.WriteString(rt)
	}
	return b.String()
}
func (n *CallableTypeNode) String() string {
	return fmt.Sprintf("CallableType(%s) @ %d:%d", n.TokenLiteral(), n.Pos.Line, n.Pos.Column)
}
