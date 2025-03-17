package ast

import (
	"fmt"
	"strings"
)

// InterfaceNode represents a PHP interface definition
type InterfaceNode struct {
	Name    string
	Methods []Node
	Pos     Position
}

func (i *InterfaceNode) NodeType() string    { return "Interface" }
func (i *InterfaceNode) GetPos() Position    { return i.Pos }
func (i *InterfaceNode) SetPos(pos Position) { i.Pos = pos }
func (i *InterfaceNode) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Interface(%s)", i.Name))
	if len(i.Methods) > 0 {
		parts = append(parts, "Methods:")
		for _, method := range i.Methods {
			parts = append(parts, "  "+method.String())
		}
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, "\n"), i.Pos.Line, i.Pos.Column)
}
func (i *InterfaceNode) TokenLiteral() string {
	return "interface"
}

// InterfaceMethodNode represents a method declaration in an interface
type InterfaceMethodNode struct {
	Name       string
	Visibility string // public, private, protected
	ReturnType string
	Params     []Node
	Pos        Position
}

func (m *InterfaceMethodNode) NodeType() string    { return "InterfaceMethod" }
func (m *InterfaceMethodNode) GetPos() Position    { return m.Pos }
func (m *InterfaceMethodNode) SetPos(pos Position) { m.Pos = pos }
func (m *InterfaceMethodNode) String() string {
	var parts []string
	if m.Visibility != "" {
		parts = append(parts, m.Visibility)
	}
	parts = append(parts, fmt.Sprintf("Method(%s)", m.Name))
	if m.ReturnType != "" {
		parts = append(parts, fmt.Sprintf(": %s", m.ReturnType))
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), m.Pos.Line, m.Pos.Column)
}
func (m *InterfaceMethodNode) TokenLiteral() string {
	return "function"
}
