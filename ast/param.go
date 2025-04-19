package ast

import (
	"fmt"
)

// ParamNode represents a function or method parameter
type ParamNode struct {
	Name         string
	TypeHint     string
	UnionType    *UnionTypeNode // Added for PHP 8.0+ union types
	DefaultValue Node
	Pos          Position
}

func (p *ParamNode) NodeType() string    { return "Param" }
func (p *ParamNode) GetPos() Position    { return p.Pos }
func (p *ParamNode) SetPos(pos Position) { p.Pos = pos }
func (p *ParamNode) String() string {
	result := fmt.Sprintf("Param($%s", p.Name)

	if p.TypeHint != "" {
		result = fmt.Sprintf("%s: %s", result, p.TypeHint)
	} else if p.UnionType != nil {
		result = fmt.Sprintf("%s: %s", result, p.UnionType.TokenLiteral())
	}

	if p.DefaultValue != nil {
		result = fmt.Sprintf("%s = %s", result, p.DefaultValue.String())
	}

	result = fmt.Sprintf("%s) @ %d:%d", result, p.Pos.Line, p.Pos.Column)
	return result
}
func (p *ParamNode) TokenLiteral() string {
	return p.Name
}
