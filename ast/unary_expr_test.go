package ast

import "testing"

type dummyNode struct{}

func (d *dummyNode) NodeType() string     { return "Dummy" }
func (d *dummyNode) GetPos() Position     { return Position{} }
func (d *dummyNode) SetPos(Position)      { /* no-op for dummy node */ }
func (d *dummyNode) String() string       { return "dummy" }
func (d *dummyNode) TokenLiteral() string { return "dummy" }

func TestUnaryExpr(t *testing.T) {
	op := "-"
	operand := &dummyNode{}
	pos := Position{Line: 1, Column: 2}
	u := &UnaryExpr{Operator: op, Operand: operand, Pos: pos}

	if u.NodeType() != "UnaryExpr" {
		t.Errorf("NodeType: got %q", u.NodeType())
	}
	if u.GetPos() != pos {
		t.Errorf("GetPos: got %+v", u.GetPos())
	}
	newPos := Position{Line: 3, Column: 4}
	u.SetPos(newPos)
	if u.GetPos() != newPos {
		t.Errorf("SetPos: got %+v", u.GetPos())
	}
	if u.String() != op+operand.String() {
		t.Errorf("String: got %q", u.String())
	}
	if u.TokenLiteral() != op {
		t.Errorf("TokenLiteral: got %q", u.TokenLiteral())
	}
}
