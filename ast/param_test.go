package ast

import "testing"

func TestParamNodeMethods(t *testing.T) {
	union := &UnionTypeNode{Types: []string{"int", "string"}, Pos: Position{Line: 4, Column: 5}}
	defaultVal := &StringLiteral{Value: "bar", Pos: Position{Line: 6, Column: 7}}
	p := &ParamNode{
		Name:         "foo",
		TypeHint:     "",
		UnionType:    union,
		DefaultValue: defaultVal,
		Visibility:   "public",
		IsPromoted:   true,
		IsVariadic:   true,
		IsByRef:      true,
		Pos:          Position{Line: 1, Column: 2},
	}
	if p.NodeType() != "Param" {
		t.Errorf("NodeType: got %q", p.NodeType())
	}
	if p.GetPos().Line != 1 || p.GetPos().Column != 2 {
		t.Errorf("GetPos: got %+v", p.GetPos())
	}
	p.SetPos(Position{Line: 3, Column: 4})
	if p.GetPos().Line != 3 || p.GetPos().Column != 4 {
		t.Errorf("SetPos: got %+v", p.GetPos())
	}
	str := p.String()
	if str == "" {
		t.Errorf("String: got %q", str)
	}
	if p.TokenLiteral() != "foo" {
		t.Errorf("TokenLiteral: got %q", p.TokenLiteral())
	}
	if p.UnionType != union {
		t.Errorf("UnionType: got %+v", p.UnionType)
	}
	if p.DefaultValue != defaultVal {
		t.Errorf("DefaultValue: got %+v", p.DefaultValue)
	}
}
