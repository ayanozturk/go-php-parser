package ast

import (
	"fmt"
	"strings"
)

// EnumNode represents a PHP enum definition
type EnumNode struct {
	Name     string
	BackedBy string // Optional type that the enum is backed by (e.g., "int", "string")
	Cases    []*EnumCaseNode
	Pos      Position
}

func (e *EnumNode) NodeType() string    { return "Enum" }
func (e *EnumNode) GetPos() Position    { return e.Pos }
func (e *EnumNode) SetPos(pos Position) { e.Pos = pos }
func (e *EnumNode) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Enum(%s)", e.Name))
	if e.BackedBy != "" {
		parts = append(parts, fmt.Sprintf(": %s", e.BackedBy))
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), e.Pos.Line, e.Pos.Column)
}
func (e *EnumNode) TokenLiteral() string {
	return "enum"
}

// EnumCaseNode represents a case in an enum
type EnumCaseNode struct {
	Name  string
	Value Node // Optional value for backed enums
	Pos   Position
}

func (e *EnumCaseNode) NodeType() string    { return "EnumCase" }
func (e *EnumCaseNode) GetPos() Position    { return e.Pos }
func (e *EnumCaseNode) SetPos(pos Position) { e.Pos = pos }
func (e *EnumCaseNode) String() string {
	if e.Value != nil {
		return fmt.Sprintf("Case(%s = %s) @ %d:%d", e.Name, e.Value.TokenLiteral(), e.Pos.Line, e.Pos.Column)
	}
	return fmt.Sprintf("Case(%s) @ %d:%d", e.Name, e.Pos.Line, e.Pos.Column)
}
func (e *EnumCaseNode) TokenLiteral() string {
	return e.Name
}
