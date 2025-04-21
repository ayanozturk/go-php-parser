package ast

import "testing"

func TestEnumNodeMethods(t *testing.T) {
	case1 := &EnumCaseNode{Name: "FOO", Value: nil, Pos: Position{Line: 2, Column: 3}}
	case2 := &EnumCaseNode{Name: "BAR", Value: &StringLiteral{Value: "bar"}, Pos: Position{Line: 3, Column: 4}}
	enum := &EnumNode{
		Name:     "MyEnum",
		BackedBy: "string",
		Cases:    []*EnumCaseNode{case1, case2},
		Pos:      Position{Line: 1, Column: 1},
	}
	if enum.NodeType() != "Enum" {
		t.Errorf("NodeType: got %q", enum.NodeType())
	}
	if enum.GetPos().Line != 1 || enum.GetPos().Column != 1 {
		t.Errorf("GetPos: got %+v", enum.GetPos())
	}
	enum.SetPos(Position{Line: 4, Column: 5})
	if enum.GetPos().Line != 4 || enum.GetPos().Column != 5 {
		t.Errorf("SetPos: got %+v", enum.GetPos())
	}
	str := enum.String()
	if str != "Enum(MyEnum) : string @ 4:5" {
		t.Errorf("String: got %q", str)
	}
	if enum.TokenLiteral() != "enum" {
		t.Errorf("TokenLiteral: got %q", enum.TokenLiteral())
	}
	if len(enum.Cases) != 2 || enum.Cases[0] != case1 || enum.Cases[1] != case2 {
		t.Errorf("Cases: got %+v", enum.Cases)
	}
}

func TestEnumCaseNodeMethods(t *testing.T) {
	caseNode := &EnumCaseNode{Name: "FOO", Value: &StringLiteral{Value: "bar"}, Pos: Position{Line: 7, Column: 8}}
	if caseNode.NodeType() != "EnumCase" {
		t.Errorf("NodeType: got %q", caseNode.NodeType())
	}
	if caseNode.GetPos().Line != 7 || caseNode.GetPos().Column != 8 {
		t.Errorf("GetPos: got %+v", caseNode.GetPos())
	}
	caseNode.SetPos(Position{Line: 9, Column: 10})
	if caseNode.GetPos().Line != 9 || caseNode.GetPos().Column != 10 {
		t.Errorf("SetPos: got %+v", caseNode.GetPos())
	}
	str := caseNode.String()
	if str != "Case(FOO = bar) @ 9:10" {
		t.Errorf("String: got %q", str)
	}
	if caseNode.TokenLiteral() != "FOO" {
		t.Errorf("TokenLiteral: got %q", caseNode.TokenLiteral())
	}
}
