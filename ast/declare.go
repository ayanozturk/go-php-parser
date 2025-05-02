package ast

import (
	"fmt"
)

type DeclareNode struct {
	Directives map[string]Node // e.g. {"strict_types": IntegerLiteral(1)}
	Pos        Position
	Body       Node // The body of the declare statement (e.g., a block or a single statement)
}

func (d *DeclareNode) NodeType() string    { return "Declare" }
func (d *DeclareNode) GetPos() Position    { return d.Pos }
func (d *DeclareNode) SetPos(pos Position) { d.Pos = pos }
func (d *DeclareNode) String() string {
	return fmt.Sprintf("declare @ %d:%d", d.Pos.Line, d.Pos.Column)
}
func (d *DeclareNode) TokenLiteral() string { return "declare" }

type DeclareDirective struct {
	Name  string // e.g. "strict_types"
	Value Node   // e.g. IntegerLiteral(1)
	Pos   Position
}

func (d *DeclareDirective) NodeType() string    { return "DeclareDirective" }
func (d *DeclareDirective) GetPos() Position    { return d.Pos }
func (d *DeclareDirective) SetPos(pos Position) { d.Pos = pos }
func (d *DeclareDirective) String() string {
	return fmt.Sprintf("DeclareDirective(%s = %s) @ %d:%d", d.Name, d.Value.String(), d.Pos.Line, d.Pos.Column)
}
func (d *DeclareDirective) TokenLiteral() string {
	return d.Name
}
