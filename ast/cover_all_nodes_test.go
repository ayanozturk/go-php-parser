package ast

import (
	"testing"
)

func TestBlockNode_EmptyStatements(t *testing.T) {
	b := &BlockNode{Statements: nil, Pos: Position{Line: 1, Column: 1}}
	_ = b.NodeType()
	_ = b.GetPos()
	b.SetPos(Position{Line: 2, Column: 2})
	_ = b.String()
	_ = b.TokenLiteral()
}

func TestIdentifier_EmptyName(t *testing.T) {
	i := &Identifier{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = i.NodeType()
	_ = i.GetPos()
	i.SetPos(Position{Line: 2, Column: 2})
	_ = i.String()
	_ = i.TokenLiteral()
}

func TestVariableNode_EmptyName(t *testing.T) {
	v := &VariableNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = v.NodeType()
	_ = v.GetPos()
	v.SetPos(Position{Line: 2, Column: 2})
	_ = v.String()
	_ = v.TokenLiteral()
}

func TestStringLiteral_EmptyValue(t *testing.T) {
	s := &StringLiteral{Value: "", Pos: Position{Line: 1, Column: 1}}
	_ = s.NodeType()
	_ = s.GetPos()
	s.SetPos(Position{Line: 2, Column: 2})
	_ = s.String()
	_ = s.TokenLiteral()
	_ = s.GetValue()
}

func TestIntegerLiteral_Zero(t *testing.T) {
	i := &IntegerLiteral{Value: 0, Pos: Position{Line: 1, Column: 1}}
	_ = i.NodeType()
	_ = i.GetPos()
	i.SetPos(Position{Line: 2, Column: 2})
	_ = i.String()
	_ = i.TokenLiteral()
	_ = i.GetValue()
}

func TestFloatLiteral_Zero(t *testing.T) {
	f := &FloatLiteral{Value: 0.0, Pos: Position{Line: 1, Column: 1}}
	_ = f.NodeType()
	_ = f.GetPos()
	f.SetPos(Position{Line: 2, Column: 2})
	_ = f.String()
	_ = f.TokenLiteral()
	_ = f.GetValue()
}

func TestInterpolatedStringLiteral_EmptyParts(t *testing.T) {
	s := &InterpolatedStringLiteral{Parts: nil, Pos: Position{Line: 1, Column: 1}}
	_ = s.NodeType()
	_ = s.GetPos()
	s.SetPos(Position{Line: 2, Column: 2})
	_ = s.String()
	_ = s.TokenLiteral()
}

func TestHeredocNode_EmptyParts(t *testing.T) {
	h := &HeredocNode{Identifier: "EOT", Parts: nil, Pos: Position{Line: 1, Column: 1}}
	_ = h.NodeType()
	_ = h.GetPos()
	h.SetPos(Position{Line: 2, Column: 2})
	_ = h.String()
	_ = h.TokenLiteral()
}

func TestYieldNode_Empty(t *testing.T) {
	y := &YieldNode{From: false, Pos: Position{Line: 1, Column: 1}}
	_ = y.NodeType()
	_ = y.GetPos()
	y.SetPos(Position{Line: 2, Column: 2})
	_ = y.String()
	_ = y.TokenLiteral()
}

func TestTypeCastNode_Empty(t *testing.T) {
	tc := &TypeCastNode{Type: "", Expr: nil, Pos: Position{Line: 1, Column: 1}}
	_ = tc.NodeType()
	_ = tc.GetPos()
	tc.SetPos(Position{Line: 2, Column: 2})
	_ = tc.String()
	_ = tc.TokenLiteral()
}

func TestUnionTypeNode_Empty(t *testing.T) {
	u := &UnionTypeNode{Types: nil, Pos: Position{Line: 1, Column: 1}}
	_ = u.NodeType()
	_ = u.GetPos()
	u.SetPos(Position{Line: 2, Column: 2})
	_ = u.String()
	_ = u.TokenLiteral()
}

func TestParamNode_Empty(t *testing.T) {
	p := &ParamNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = p.NodeType()
	_ = p.GetPos()
	p.SetPos(Position{Line: 2, Column: 2})
	_ = p.String()
	_ = p.TokenLiteral()
}

func TestFunctionNode_Empty(t *testing.T) {
	f := &FunctionNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = f.NodeType()
	_ = f.GetPos()
	f.SetPos(Position{Line: 2, Column: 2})
	_ = f.String()
	_ = f.TokenLiteral()
}

func TestFunctionCallNode_Empty(t *testing.T) {
	f := &FunctionCallNode{Name: nil, Args: nil, Pos: Position{Line: 1, Column: 1}}
	_ = f.NodeType()
	_ = f.GetPos()
	f.SetPos(Position{Line: 2, Column: 2})
	_ = f.String()
	_ = f.TokenLiteral()
}

func TestUnpackedArgumentNode_Empty(t *testing.T) {
	u := &UnpackedArgumentNode{Expr: nil, Pos: Position{Line: 1, Column: 1}}
	_ = u.NodeType()
	_ = u.GetPos()
	u.SetPos(Position{Line: 2, Column: 2})
	_ = u.String()
	_ = u.TokenLiteral()
}

func TestClassNode_Empty(t *testing.T) {
	c := &ClassNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = c.NodeType()
	_ = c.GetPos()
	c.SetPos(Position{Line: 2, Column: 2})
	_ = c.String()
	_ = c.TokenLiteral()
}

func TestPropertyNode_Empty(t *testing.T) {
	p := &PropertyNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = p.NodeType()
	_ = p.GetPos()
	p.SetPos(Position{Line: 2, Column: 2})
	_ = p.String()
	_ = p.TokenLiteral()
}

func TestEnumNode_Empty(t *testing.T) {
	e := &EnumNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = e.NodeType()
	_ = e.GetPos()
	e.SetPos(Position{Line: 2, Column: 2})
	_ = e.String()
	_ = e.TokenLiteral()
}

func TestEnumCaseNode_Empty(t *testing.T) {
	ec := &EnumCaseNode{Name: "", Value: nil, Pos: Position{Line: 1, Column: 1}}
	_ = ec.NodeType()
	_ = ec.GetPos()
	ec.SetPos(Position{Line: 2, Column: 2})
	_ = ec.String()
	_ = ec.TokenLiteral()
}

func TestInterfaceNode_Empty(t *testing.T) {
	i := &InterfaceNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = i.NodeType()
	_ = i.GetPos()
	i.SetPos(Position{Line: 2, Column: 2})
	_ = i.String()
	_ = i.TokenLiteral()
}

func TestInterfaceMethodNode_Empty(t *testing.T) {
	m := &InterfaceMethodNode{Name: "", Pos: Position{Line: 1, Column: 1}}
	_ = m.NodeType()
	_ = m.GetPos()
	m.SetPos(Position{Line: 2, Column: 2})
	_ = m.String()
	_ = m.TokenLiteral()
}

func TestCommentNode_Empty(t *testing.T) {
	c := &CommentNode{Value: "", Pos: Position{Line: 1, Column: 1}}
	_ = c.NodeType()
	_ = c.GetPos()
	c.SetPos(Position{Line: 2, Column: 2})
	_ = c.String()
	_ = c.TokenLiteral()
}

func TestClassConstFetchNode_Empty(t *testing.T) {
	cc := &ClassConstFetchNode{Class: "", Const: "", Pos: Position{Line: 1, Column: 1}}
	_ = cc.NodeType()
	_ = cc.GetPos()
	cc.SetPos(Position{Line: 2, Column: 2})
	_ = cc.String()
	_ = cc.TokenLiteral()
}
