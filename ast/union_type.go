package ast

import (
	"fmt"
	"strings"
)

// UnionTypeNode represents a PHP 8.0+ union type declaration
type UnionTypeNode struct {
	Types  []string // List of type names in the union
	Pos    Position
	EndPos Position
}

func (u *UnionTypeNode) NodeType() string       { return "UnionType" }
func (u *UnionTypeNode) GetPos() Position       { return u.Pos }
func (u *UnionTypeNode) SetPos(pos Position)    { u.Pos = pos }
func (u *UnionTypeNode) GetEndPos() Position    { return u.EndPos }
func (u *UnionTypeNode) SetEndPos(pos Position) { u.EndPos = pos }
func (u *UnionTypeNode) String() string {
	return fmt.Sprintf("UnionType(%s) @ %d:%d", strings.Join(u.Types, "|"), u.Pos.Line, u.Pos.Column)
}
func (u *UnionTypeNode) TokenLiteral() string {
	return strings.Join(u.Types, "|")
}
