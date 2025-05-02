package ast

import (
	"fmt"
	"strings"
)

// ClassNode represents a PHP class definition
type ClassNode struct {
	Name       string
	Extends    string
	Implements []string
	Properties []Node
	Methods    []Node
	Pos        Position
	Modifier   string // final, abstract, or ""
}

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

// NewNode represents object instantiation
type NewNode struct {
	ClassName string
	Args      []Node
	Pos       Position
}

func (n *NewNode) NodeType() string    { return "New" }
func (n *NewNode) GetPos() Position    { return n.Pos }
func (n *NewNode) SetPos(pos Position) { n.Pos = pos }
func (n *NewNode) String() string {
	return fmt.Sprintf("New(%s) @ %d:%d", n.ClassName, n.Pos.Line, n.Pos.Column)
}
func (n *NewNode) TokenLiteral() string {
	return "new"
}

// MethodCallNode represents a method call on an object
type MethodCallNode struct {
	Object Node
	Method string
	Args   []Node
	Pos    Position
}

func (m *MethodCallNode) NodeType() string    { return "MethodCall" }
func (m *MethodCallNode) GetPos() Position    { return m.Pos }
func (m *MethodCallNode) SetPos(pos Position) { m.Pos = pos }
func (m *MethodCallNode) String() string {
	return fmt.Sprintf("MethodCall(%s) @ %d:%d", m.Method, m.Pos.Line, m.Pos.Column)
}
func (m *MethodCallNode) TokenLiteral() string {
	return m.Method
}

// TraitNode represents a trait definition
type TraitNode struct {
	Name *Identifier // The name of the trait
	Body []Node      // Statements within the trait block (methods, properties)
	Pos  Position    // The position of the 'trait' keyword
}

func (t *TraitNode) NodeType() string    { return "Trait" }
func (t *TraitNode) GetPos() Position    { return t.Pos }
func (t *TraitNode) SetPos(pos Position) { t.Pos = pos }
func (t *TraitNode) String() string {
	return fmt.Sprintf("Trait(%s) @ %d:%d", t.Name.String(), t.Pos.Line, t.Pos.Column)
}
func (t *TraitNode) TokenLiteral() string {
	return "trait"
}
