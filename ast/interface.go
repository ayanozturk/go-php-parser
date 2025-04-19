package ast

import (
	"fmt"
	"strings"
)

// InterfaceNode represents a PHP interface definition
type InterfaceNode struct {
	Name    string
	Extends []string
	Methods []Node
	Pos     Position
}

func (i *InterfaceNode) NodeType() string    { return "Interface" }
func (i *InterfaceNode) GetPos() Position    { return i.Pos }
func (i *InterfaceNode) SetPos(pos Position) { i.Pos = pos }
func (i *InterfaceNode) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Interface(%s)", i.Name))
	if len(i.Extends) > 0 {
		parts = append(parts, fmt.Sprintf("Extends: %s", strings.Join(i.Extends, ", ")))
	}
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
	parts = append(parts, fmt.Sprintf("function %s(", m.Name))

	// Add parameters
	paramStrs := make([]string, len(m.Params))
	for i, param := range m.Params {
		paramStrs[i] = param.String()
	}
	parts = append(parts, strings.Join(paramStrs, ", ")+")")

	// Add return type if present
	if m.ReturnType != "" {
		parts = append(parts, ": "+m.ReturnType)
	}

	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), m.Pos.Line, m.Pos.Column)
}
func (m *InterfaceMethodNode) TokenLiteral() string {
	return "function"
}
