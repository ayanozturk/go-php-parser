package ast

import (
	"fmt"
	"strings"
)

// ParamNode represents a function or method parameter
type ParamNode struct {
	Name         string
	TypeHint     string
	UnionType    *UnionTypeNode // For PHP 8.0+ union types
	DefaultValue Node
	Visibility   string // public, protected, private (for promoted constructor params)
	IsPromoted   bool   // true if this param is promoted to a property
	IsVariadic   bool   // true if this param is variadic (...$values)
	IsByRef      bool   // true if this param is passed by reference (&$data)
	Pos          Position
}

func (p *ParamNode) NodeType() string    { return "Param" }
func (p *ParamNode) GetPos() Position    { return p.Pos }
func (p *ParamNode) SetPos(pos Position) { p.Pos = pos }
func (p *ParamNode) String() string {
	var parts []string
	if p.Visibility != "" {
		parts = append(parts, p.Visibility)
	}
	if p.TypeHint != "" {
		parts = append(parts, p.TypeHint)
	} else if p.UnionType != nil {
		parts = append(parts, p.UnionType.TokenLiteral())
	}
	if p.IsByRef {
		parts = append(parts, "&")
	}
	parts = append(parts, "$"+p.Name)
	if p.DefaultValue != nil {
		parts = append(parts, "=", p.DefaultValue.String())
	}
	if p.IsPromoted {
		parts = append(parts, "[promoted]")
	}
	if p.IsVariadic {
		parts = append(parts, "[variadic]")
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), p.Pos.Line, p.Pos.Column)
}
func (p *ParamNode) TokenLiteral() string {
	return p.Name
}
