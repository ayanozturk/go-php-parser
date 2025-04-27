package ast

import (
	"fmt"
	"strings"
)

// UnionTypeNode represents a PHP 8.0+ union type declaration
type UnionTypeNode struct {
	Types []string // List of type names in the union
	Pos   Position
}

func (u *UnionTypeNode) NodeType() string    { return "UnionType" }
func (u *UnionTypeNode) GetPos() Position    { return u.Pos }
func (u *UnionTypeNode) SetPos(pos Position) { u.Pos = pos }
func (u *UnionTypeNode) String() string {
	return fmt.Sprintf("UnionType(%s) @ %d:%d", strings.Join(u.Types, "|"), u.Pos.Line, u.Pos.Column)
}
func (u *UnionTypeNode) TokenLiteral() string {
	return strings.Join(u.Types, "|")
}

// IntersectionTypeNode represents a PHP 8.1+ intersection type declaration
// e.g. Foo&Bar&Baz
// Types: list of type names (strings)
type IntersectionTypeNode struct {
	Types []string // List of type names in the intersection
	Pos   Position
}

func (i *IntersectionTypeNode) NodeType() string    { return "IntersectionType" }
func (i *IntersectionTypeNode) GetPos() Position    { return i.Pos }
func (i *IntersectionTypeNode) SetPos(pos Position) { i.Pos = pos }
func (i *IntersectionTypeNode) String() string {
	return fmt.Sprintf("IntersectionType(%s) @ %d:%d", strings.Join(i.Types, "&"), i.Pos.Line, i.Pos.Column)
}
func (i *IntersectionTypeNode) TokenLiteral() string {
	return strings.Join(i.Types, "&")
}
