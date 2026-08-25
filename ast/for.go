package ast

import "fmt"

// ForNode represents a PHP for loop. Each control clause is an expression
// list because PHP permits comma-separated initializers, conditions, and
// updates. Empty clauses are represented by empty slices.
type ForNode struct {
	Init       []Node
	Conditions []Node
	Updates    []Node
	Body       []Node
	Pos        Position
	EndPos     Position
}

func (f *ForNode) NodeType() string       { return "For" }
func (f *ForNode) GetPos() Position       { return f.Pos }
func (f *ForNode) SetPos(pos Position)    { f.Pos = pos }
func (f *ForNode) GetEndPos() Position    { return f.EndPos }
func (f *ForNode) SetEndPos(pos Position) { f.EndPos = pos }
func (f *ForNode) String() string {
	return fmt.Sprintf("For(init: %d, conditions: %d, updates: %d) @ %d:%d", len(f.Init), len(f.Conditions), len(f.Updates), f.Pos.Line, f.Pos.Column)
}
func (f *ForNode) TokenLiteral() string { return "for" }
