package ast

import "strings"

// IntersectionTypeNode represents a PHP 8.1+ intersection type.
// Types holds nested type/name nodes (not string spellings).
type IntersectionTypeNode struct {
	Types  []Node
	Pos    Position
	EndPos Position
}

func (i *IntersectionTypeNode) NodeType() string       { return "IntersectionType" }
func (i *IntersectionTypeNode) GetPos() Position       { return i.Pos }
func (i *IntersectionTypeNode) SetPos(pos Position)    { i.Pos = pos }
func (i *IntersectionTypeNode) GetEndPos() Position    { return i.EndPos }
func (i *IntersectionTypeNode) SetEndPos(pos Position) { i.EndPos = pos }
func (i *IntersectionTypeNode) TokenLiteral() string {
	return joinTypeNodes(i.Types, "&")
}

func (i *IntersectionTypeNode) String() string {
	parts := make([]string, 0, len(i.Types))
	for _, n := range i.Types {
		parts = append(parts, TypeText(n))
	}
	return strings.Join(parts, " & ")
}
