package ast

import (
	"fmt"
	"strings"
)

// ParamNode represents a function or method parameter
type ParamNode struct {
	Name         string
	TypeHint     Node // IdentifierNode | UnionTypeNode | IntersectionTypeNode
	DefaultValue Node
	Modifiers    ModifierList // promoted-property visibility / asymmetric / readonly
	IsPromoted   bool         // true if this param is promoted to a property
	IsReadonly   bool         // true if this promoted parameter is readonly
	IsVariadic   bool         // true if this param is variadic (...$values)
	IsByRef      bool         // true if this param is passed by reference (&$data)
	PHPDoc       *PHPDocNode
	Pos          Position
	EndPos       Position
}

func (p *ParamNode) NodeType() string       { return "Param" }
func (p *ParamNode) GetPos() Position       { return p.Pos }
func (p *ParamNode) SetPos(pos Position)    { p.Pos = pos }
func (p *ParamNode) GetEndPos() Position    { return p.EndPos }
func (p *ParamNode) SetEndPos(pos Position) { p.EndPos = pos }
func (p *ParamNode) String() string {
	var parts []string
	if s := p.Modifiers.String(); s != "" {
		parts = append(parts, s)
	}
	if text := TypeText(p.TypeHint); text != "" {
		parts = append(parts, text)
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
