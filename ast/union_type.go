package ast

import (
	"fmt"
	"strings"
)

// UnionTypeNode represents a PHP 8.0+ union type declaration.
// Types holds nested type/name nodes (not string spellings).
type UnionTypeNode struct {
	Types  []Node
	Pos    Position
	EndPos Position
}

func (u *UnionTypeNode) NodeType() string       { return "UnionType" }
func (u *UnionTypeNode) GetPos() Position       { return u.Pos }
func (u *UnionTypeNode) SetPos(pos Position)    { u.Pos = pos }
func (u *UnionTypeNode) GetEndPos() Position    { return u.EndPos }
func (u *UnionTypeNode) SetEndPos(pos Position) { u.EndPos = pos }
func (u *UnionTypeNode) String() string {
	return fmt.Sprintf("UnionType(%s) @ %d:%d", joinTypeNodes(u.Types, "|"), u.Pos.Line, u.Pos.Column)
}
func (u *UnionTypeNode) TokenLiteral() string {
	return joinTypeNodes(u.Types, "|")
}

func joinTypeNodes(nodes []Node, sep string) string {
	if len(nodes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, TypeText(n))
	}
	return strings.Join(parts, sep)
}
