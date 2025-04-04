package ast

import (
	"fmt"
	"strings"
)

// FunctionNode represents a PHP function definition
type FunctionNode struct {
	Name       string
	Visibility string // public, private, protected
	ReturnType string
	Params     []Node
	Body       []Node
	Pos        Position
}

func (f *FunctionNode) NodeType() string    { return "Function" }
func (f *FunctionNode) GetPos() Position    { return f.Pos }
func (f *FunctionNode) SetPos(pos Position) { f.Pos = pos }
func (f *FunctionNode) String() string {
	var parts []string
	if f.Visibility != "" {
		parts = append(parts, f.Visibility)
	}
	parts = append(parts, fmt.Sprintf("Function(%s)", f.Name))
	if f.ReturnType != "" {
		parts = append(parts, fmt.Sprintf(": %s", f.ReturnType))
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), f.Pos.Line, f.Pos.Column)
}
func (f *FunctionNode) TokenLiteral() string {
	return "function"
}

// ParameterNode represents a function parameter
type ParameterNode struct {
	Name         string
	TypeHint     string // Type hint for the parameter (e.g., string, int, array)
	DefaultValue Node   // Optional default value
	Pos          Position
}

func (p *ParameterNode) NodeType() string    { return "Parameter" }
func (p *ParameterNode) GetPos() Position    { return p.Pos }
func (p *ParameterNode) SetPos(pos Position) { p.Pos = pos }
func (p *ParameterNode) String() string {
	var parts []string
	if p.TypeHint != "" {
		parts = append(parts, p.TypeHint)
	}
	parts = append(parts, p.Name)
	if p.DefaultValue != nil {
		parts = append(parts, "=", p.DefaultValue.String())
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), p.Pos.Line, p.Pos.Column)
}
func (p *ParameterNode) TokenLiteral() string {
	return p.Name
}
