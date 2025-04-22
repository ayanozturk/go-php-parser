package ast

import "testing"

func TestInterpolatedStringLiteralNode(t *testing.T) {
	part1 := &StringLiteral{Value: "foo", Pos: Position{Line: 1, Column: 2}}
	part2 := &VariableNode{Name: "bar", Pos: Position{Line: 1, Column: 5}}
	n := &InterpolatedStringLiteral{
		Parts: []Node{part1, part2},
		Pos:   Position{Line: 1, Column: 1},
	}
	if n.NodeType() != "InterpolatedString" {
		t.Errorf("NodeType: got %q", n.NodeType())
	}
	if n.GetPos().Line != 1 || n.GetPos().Column != 1 {
		t.Errorf("GetPos: got %+v", n.GetPos())
	}
	n.SetPos(Position{Line: 2, Column: 3})
	if n.GetPos().Line != 2 || n.GetPos().Column != 3 {
		t.Errorf("SetPos: got %+v", n.GetPos())
	}
	if n.String() == "" {
		t.Errorf("String: got %q", n.String())
	}
	if n.TokenLiteral() != "foobar" {
		t.Errorf("TokenLiteral: got %q", n.TokenLiteral())
	}
	if len(n.Parts) != 2 || n.Parts[0] != part1 || n.Parts[1] != part2 {
		t.Errorf("Parts: got %+v", n.Parts)
	}
}

func TestFloatLiteralNodeEdgeCases(t *testing.T) {
	f := &FloatLiteral{Value: 0.0, Pos: Position{Line: 1, Column: 1}}
	if f.NodeType() != "FloatLiteral" {
		t.Errorf("NodeType: got %q", f.NodeType())
	}
	if f.GetPos().Line != 1 || f.GetPos().Column != 1 {
		t.Errorf("GetPos: got %+v", f.GetPos())
	}
	f.SetPos(Position{Line: 2, Column: 2})
	if f.GetPos().Line != 2 || f.GetPos().Column != 2 {
		t.Errorf("SetPos: got %+v", f.GetPos())
	}
	if f.String() == "" {
		t.Errorf("String: got %q", f.String())
	}
	if f.TokenLiteral() != "0" && f.TokenLiteral() != "0.0" {
		t.Errorf("TokenLiteral: got %q", f.TokenLiteral())
	}
	if val, ok := f.GetValue().(float64); !ok || val != 0.0 {
		t.Errorf("GetValue: got %v", f.GetValue())
	}
}

func TestUnionTypeNodeEmptyTypes(t *testing.T) {
	u := &UnionTypeNode{Types: []string{}, Pos: Position{Line: 1, Column: 1}}
	if u.NodeType() != "UnionType" {
		t.Errorf("NodeType: got %q", u.NodeType())
	}
	if u.TokenLiteral() != "" {
		t.Errorf("TokenLiteral: got %q", u.TokenLiteral())
	}
	if u.String() != "UnionType() @ 1:1" {
		t.Errorf("String: got %q", u.String())
	}
}

func TestParamNodeNilFields(t *testing.T) {
	p := &ParamNode{Name: "foo", Pos: Position{Line: 1, Column: 1}}
	if p.NodeType() != "Param" {
		t.Errorf("NodeType: got %q", p.NodeType())
	}
	if p.UnionType != nil {
		t.Errorf("UnionType: got %+v", p.UnionType)
	}
	if p.DefaultValue != nil {
		t.Errorf("DefaultValue: got %+v", p.DefaultValue)
	}
	if p.String() == "" {
		t.Errorf("String: got %q", p.String())
	}
	if p.TokenLiteral() != "foo" {
		t.Errorf("TokenLiteral: got %q", p.TokenLiteral())
	}
}

func TestYieldNodeNilFields(t *testing.T) {
	y := &YieldNode{From: false, Pos: Position{Line: 1, Column: 1}}
	if y.NodeType() != "Yield" {
		t.Errorf("NodeType: got %q", y.NodeType())
	}
	if y.Key != nil || y.Value != nil {
		t.Errorf("Key/Value should be nil: got %+v %+v", y.Key, y.Value)
	}
	if y.String() != "yield @ 1:1" {
		t.Errorf("String: got %q", y.String())
	}
	if y.TokenLiteral() != "yield" {
		t.Errorf("TokenLiteral: got %q", y.TokenLiteral())
	}
}

func TestTypeCastNodeNilExpr(t *testing.T) {
	tc := &TypeCastNode{Type: "int", Pos: Position{Line: 1, Column: 1}}
	if tc.NodeType() != "TypeCast" {
		t.Errorf("NodeType: got %q", tc.NodeType())
	}
	if tc.Expr != nil {
		t.Errorf("Expr should be nil: got %+v", tc.Expr)
	}
	if tc.String() != "(int) @ 1:1" {
		t.Errorf("String: got %q", tc.String())
	}
	if tc.TokenLiteral() != "int" {
		t.Errorf("TokenLiteral: got %q", tc.TokenLiteral())
	}
}
