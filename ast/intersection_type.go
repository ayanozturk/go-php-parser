package ast

import "strings"

type IntersectionTypeNode struct {
	Types []string // List of type names in the intersection
	Pos   Position
}

func (i *IntersectionTypeNode) NodeType() string    { return "IntersectionType" }
func (i *IntersectionTypeNode) GetPos() Position    { return i.Pos }
func (i *IntersectionTypeNode) SetPos(pos Position) { i.Pos = pos }
func (i *IntersectionTypeNode) TokenLiteral() string {
	return "&"
}

func (i *IntersectionTypeNode) String() string {
	return strings.Join(i.Types, " & ")
}
