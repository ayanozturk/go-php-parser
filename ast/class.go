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
	Constants  []Node // Class constants
	Pos        Position
	EndPos     Position
	Modifier   string      // final, abstract, or ""
	PHPDoc     *PHPDocNode // Associated PHPDoc comment
}

func (c *ClassNode) NodeType() string       { return "Class" }
func (c *ClassNode) GetPos() Position       { return c.Pos }
func (c *ClassNode) SetPos(pos Position)    { c.Pos = pos }
func (c *ClassNode) GetEndPos() Position    { return c.EndPos }
func (c *ClassNode) SetEndPos(pos Position) { c.EndPos = pos }
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
	Name          string
	TypeHint      string
	PHPDoc        *PHPDocNode
	DefaultValue  Node
	Visibility    string // public, private, protected
	SetVisibility string // PHP 8.4 asymmetric visibility: public(set), protected(set), private(set)
	IsStatic      bool
	IsReadonly    bool
	Hooks         []PropertyHookNode
	Pos           Position
	EndPos        Position
}

type PropertyHookNode struct {
	Name      string
	IsByRef   bool
	Parameter string
	Expr      Node
	Body      []Node
	Pos       Position
	EndPos    Position
}

func (n *PropertyNode) GetPos() Position {
	return n.Pos
}

func (n *PropertyNode) SetPos(pos Position) {
	n.Pos = pos
}

func (n *PropertyNode) GetEndPos() Position {
	return n.EndPos
}

func (n *PropertyNode) SetEndPos(pos Position) {
	n.EndPos = pos
}

func (n *PropertyNode) NodeType() string {
	return "Property"
}

func (n *PropertyNode) String() string {
	var parts []string
	if n.Visibility != "" {
		parts = append(parts, n.Visibility)
	}
	if n.SetVisibility != "" {
		parts = append(parts, n.SetVisibility+"(set)")
	}
	parts = append(parts, fmt.Sprintf("Property($%s)", n.Name))
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), n.Pos.Line, n.Pos.Column)
}

func (n *PropertyNode) TokenLiteral() string {
	return n.Name
}

type TraitUseNode struct {
	Traits      []string
	Adaptations []TraitAdaptation
	Pos         Position
	EndPos      Position
}

// TraitAdaptation represents a single entry inside a trait `use { ... }`
// adaptation block, e.g. `A::foo as bar;` or `A::foo insteadof B;`.
type TraitAdaptation struct {
	// Trait is the (optional) trait name qualifying Method, e.g. "A" in
	// "A::foo as bar;". Empty when the method is referenced unqualified.
	Trait string
	// Method is the trait method being aliased or resolved.
	Method string
	// As is the new alias name, when this is an "as" adaptation. Empty
	// otherwise.
	As string
	// Visibility is an optional visibility modifier on an "as" adaptation
	// (e.g. "protected" in "A::foo as protected bar;"). Empty if absent.
	Visibility string
	// InsteadOf lists the trait names losing precedence for Method, when
	// this is an "insteadof" adaptation. Empty otherwise.
	InsteadOf []string
}

func (t *TraitUseNode) NodeType() string       { return "TraitUse" }
func (t *TraitUseNode) GetPos() Position       { return t.Pos }
func (t *TraitUseNode) SetPos(pos Position)    { t.Pos = pos }
func (t *TraitUseNode) GetEndPos() Position    { return t.EndPos }
func (t *TraitUseNode) SetEndPos(pos Position) { t.EndPos = pos }
func (t *TraitUseNode) String() string {
	return fmt.Sprintf("TraitUse(%s) @ %d:%d", strings.Join(t.Traits, ", "), t.Pos.Line, t.Pos.Column)
}
func (t *TraitUseNode) TokenLiteral() string { return "use" }

// NewNode represents object instantiation
type NewNode struct {
	ClassName string
	ClassExpr Node
	Args      []Node
	Pos       Position
	EndPos    Position
}

func (n *NewNode) NodeType() string       { return "New" }
func (n *NewNode) GetPos() Position       { return n.Pos }
func (n *NewNode) SetPos(pos Position)    { n.Pos = pos }
func (n *NewNode) GetEndPos() Position    { return n.EndPos }
func (n *NewNode) SetEndPos(pos Position) { n.EndPos = pos }
func (n *NewNode) String() string {
	className := n.ClassName
	if n.ClassExpr != nil {
		className = n.ClassExpr.String()
	}
	return fmt.Sprintf("New(%s) @ %d:%d", className, n.Pos.Line, n.Pos.Column)
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
	EndPos Position
}

func (m *MethodCallNode) NodeType() string       { return "MethodCall" }
func (m *MethodCallNode) GetPos() Position       { return m.Pos }
func (m *MethodCallNode) SetPos(pos Position)    { m.Pos = pos }
func (m *MethodCallNode) GetEndPos() Position    { return m.EndPos }
func (m *MethodCallNode) SetEndPos(pos Position) { m.EndPos = pos }
func (m *MethodCallNode) String() string {
	return fmt.Sprintf("MethodCall(%s) @ %d:%d", m.Method, m.Pos.Line, m.Pos.Column)
}
func (m *MethodCallNode) TokenLiteral() string {
	return m.Method
}

// TraitNode represents a trait definition
type TraitNode struct {
	Name   *Identifier // The name of the trait
	Body   []Node      // Statements within the trait block (methods, properties)
	Pos    Position    // The position of the 'trait' keyword
	EndPos Position
}

func (t *TraitNode) NodeType() string       { return "Trait" }
func (t *TraitNode) GetPos() Position       { return t.Pos }
func (t *TraitNode) SetPos(pos Position)    { t.Pos = pos }
func (t *TraitNode) GetEndPos() Position    { return t.EndPos }
func (t *TraitNode) SetEndPos(pos Position) { t.EndPos = pos }
func (t *TraitNode) String() string {
	return fmt.Sprintf("Trait(%s) @ %d:%d", t.Name.String(), t.Pos.Line, t.Pos.Column)
}
func (t *TraitNode) TokenLiteral() string {
	return "trait"
}
