package ast

import (
	"fmt"
	"strings"
)

func (c *ClassNode) NodeType() string    { return "Class" }
func (c *ClassNode) GetPos() Position    { return c.Pos }
func (c *ClassNode) SetPos(pos Position) { c.Pos = pos }
func (c *ClassNode) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Class(%s)", c.Name))
	if c.Extends != "" {
		parts = append(parts, fmt.Sprintf("extends %s", c.Extends))
	}
	if len(c.Implements) > 0 {
		parts = append(parts, fmt.Sprintf("implements %s", strings.Join(c.Implements, ", ")))
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), c.Pos.Line, c.Pos.Column)
}
func (c *ClassNode) TokenLiteral() string {
	return "class"
}

// PropertyNode represents a class property
type PropertyNode struct {
	Name         string
	TypeHint     string
	DefaultValue Node
	Visibility   string // public, private, protected
	IsStatic     bool
	IsReadonly   bool
	Pos          Position
}

func (n *PropertyNode) GetPos() Position {
	return n.Pos
}

func (n *PropertyNode) SetPos(pos Position) {
	n.Pos = pos
}

func (n *PropertyNode) NodeType() string {
	return "Property"
}

func (n *PropertyNode) String() string {
	var parts []string
	if n.Visibility != "" {
		parts = append(parts, n.Visibility)
	}
	parts = append(parts, fmt.Sprintf("Property($%s)", n.Name))
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), n.Pos.Line, n.Pos.Column)
}

func (n *PropertyNode) TokenLiteral() string {
	return n.Name
}
