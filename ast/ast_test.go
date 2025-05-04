package ast

import (
	"testing"
)

const (
	errNodeType     = "NodeType: got %q"
	errGetPos       = "GetPos: got %+v"
	errSetPos       = "SetPos: got %+v"
	errStringEmpty  = "String should not be empty"
	errTokenLiteral = "TokenLiteral: got %q"
	errGetValue     = "GetValue: got %v"
)

func TestPositionStruct(t *testing.T) {
	pos := Position{Line: 1, Column: 2, Offset: 3}
	if pos.Line != 1 || pos.Column != 2 || pos.Offset != 3 {
		t.Errorf("unexpected Position values: %+v", pos)
	}
}

func TestIdentifierNode(t *testing.T) {
	id := &Identifier{Name: "foo", Pos: Position{Line: 1, Column: 2}}
	if id.NodeType() != "Identifier" {
		t.Errorf(errNodeType, id.NodeType())
	}
	if id.GetPos().Line != 1 || id.GetPos().Column != 2 {
		t.Errorf(errGetPos, id.GetPos())
	}
	id.SetPos(Position{Line: 3, Column: 4})
	if id.GetPos().Line != 3 || id.GetPos().Column != 4 {
		t.Errorf(errSetPos, id.GetPos())
	}
	if id.String() == "" {
		t.Error(errStringEmpty)
	}
	if id.TokenLiteral() != "foo" {
		t.Errorf(errTokenLiteral, id.TokenLiteral())
	}
}

func TestVariableNode(t *testing.T) {
	v := &VariableNode{Name: "bar", Pos: Position{Line: 2, Column: 3}}
	if v.NodeType() != "Variable" {
		t.Errorf(errNodeType, v.NodeType())
	}
	if v.GetPos().Line != 2 || v.GetPos().Column != 3 {
		t.Errorf(errGetPos, v.GetPos())
	}
	v.SetPos(Position{Line: 4, Column: 5})
	if v.GetPos().Line != 4 || v.GetPos().Column != 5 {
		t.Errorf(errSetPos, v.GetPos())
	}
	if v.String() == "" {
		t.Error(errStringEmpty)
	}
	if v.TokenLiteral() != "bar" {
		t.Errorf(errTokenLiteral, v.TokenLiteral())
	}
}

func TestStringLiteralNode(t *testing.T) {
	s := &StringLiteral{Value: "baz", Pos: Position{Line: 3, Column: 4}}
	if s.NodeType() != "StringLiteral" {
		t.Errorf(errNodeType, s.NodeType())
	}
	if s.GetPos().Line != 3 || s.GetPos().Column != 4 {
		t.Errorf(errGetPos, s.GetPos())
	}
	s.SetPos(Position{Line: 5, Column: 6})
	if s.GetPos().Line != 5 || s.GetPos().Column != 6 {
		t.Errorf(errSetPos, s.GetPos())
	}
	if s.String() == "" {
		t.Error(errStringEmpty)
	}
	if s.TokenLiteral() != "baz" {
		t.Errorf(errTokenLiteral, s.TokenLiteral())
	}
	if val, ok := s.GetValue().(string); !ok || val != "baz" {
		t.Errorf(errGetValue, s.GetValue())
	}
}

func TestIntegerLiteralNode(t *testing.T) {
	i := &IntegerLiteral{Value: 42, Pos: Position{Line: 6, Column: 7}}
	if i.NodeType() != "IntegerLiteral" {
		t.Errorf(errNodeType, i.NodeType())
	}
	if i.GetPos().Line != 6 || i.GetPos().Column != 7 {
		t.Errorf(errGetPos, i.GetPos())
	}
	i.SetPos(Position{Line: 8, Column: 9})
	if i.GetPos().Line != 8 || i.GetPos().Column != 9 {
		t.Errorf(errSetPos, i.GetPos())
	}
	if i.String() == "" {
		t.Error(errStringEmpty)
	}
	if i.TokenLiteral() != "42" {
		t.Errorf(errTokenLiteral, i.TokenLiteral())
	}
	if val, ok := i.GetValue().(int64); !ok || val != 42 {
		t.Errorf(errGetValue, i.GetValue())
	}
}

func TestFloatLiteralNode(t *testing.T) {
	f := &FloatLiteral{Value: 3.14, Pos: Position{Line: 10, Column: 11}}
	if f.NodeType() != "FloatLiteral" {
		t.Errorf(errNodeType, f.NodeType())
	}
	if f.GetPos().Line != 10 || f.GetPos().Column != 11 {
		t.Errorf(errGetPos, f.GetPos())
	}
	f.SetPos(Position{Line: 12, Column: 13})
	if f.GetPos().Line != 12 || f.GetPos().Column != 13 {
		t.Errorf(errSetPos, f.GetPos())
	}
	if f.String() == "" {
		t.Error(errStringEmpty)
	}
	if f.TokenLiteral() != "3.14" {
		t.Errorf(errTokenLiteral, f.TokenLiteral())
	}
	if val, ok := f.GetValue().(float64); !ok || val != 3.14 {
		t.Errorf(errGetValue, f.GetValue())
	}
}

func TestBlockNode(t *testing.T) {
	stmt := &Identifier{Name: "stmt", Pos: Position{Line: 1, Column: 1}}
	block := &BlockNode{Statements: []Node{stmt}, Pos: Position{Line: 2, Column: 2}}
	if block.NodeType() != "Block" {
		t.Errorf(errNodeType, block.NodeType())
	}
	if block.GetPos().Line != 2 || block.GetPos().Column != 2 {
		t.Errorf(errGetPos, block.GetPos())
	}
	block.SetPos(Position{Line: 3, Column: 3})
	if block.GetPos().Line != 3 || block.GetPos().Column != 3 {
		t.Errorf(errSetPos, block.GetPos())
	}
	if block.String() == "" {
		t.Error(errStringEmpty)
	}
	if block.TokenLiteral() != "{" {
		t.Errorf(errTokenLiteral, block.TokenLiteral())
	}
	if len(block.Statements) != 1 || block.Statements[0] != stmt {
		t.Errorf("Statements: got %+v", block.Statements)
	}
}

func TestTypeCastNode(t *testing.T) {
	typeCast := &TypeCastNode{Type: "int", Expr: &Identifier{Name: "foo"}, Pos: Position{Line: 1, Column: 1}}
	if typeCast.NodeType() != "TypeCast" {
		t.Errorf(errNodeType, typeCast.NodeType())
	}
	if typeCast.GetPos().Line != 1 {
		t.Errorf(errGetPos, typeCast.GetPos())
	}
	typeCast.SetPos(Position{Line: 2, Column: 2})
	if typeCast.GetPos().Line != 2 {
		t.Errorf(errSetPos, typeCast.GetPos())
	}
	if typeCast.String() == "" {
		t.Error(errStringEmpty)
	}
	if typeCast.TokenLiteral() != "int" {
		t.Errorf(errTokenLiteral, typeCast.TokenLiteral())
	}
}

func TestYieldNode(t *testing.T) {
	yield := &YieldNode{Key: nil, Value: &Identifier{Name: "foo"}, From: true, Pos: Position{Line: 1, Column: 1}}
	if yield.NodeType() != "Yield" {
		t.Errorf(errNodeType, yield.NodeType())
	}
	if yield.GetPos().Line != 1 {
		t.Errorf(errGetPos, yield.GetPos())
	}
	yield.SetPos(Position{Line: 2, Column: 2})
	if yield.GetPos().Line != 2 {
		t.Errorf(errSetPos, yield.GetPos())
	}
	if yield.String() == "" {
		t.Error(errStringEmpty)
	}
	if yield.TokenLiteral() != "yield" {
		t.Errorf(errTokenLiteral, yield.TokenLiteral())
	}
}

func TestHeredocNode(t *testing.T) {
	heredoc := &HeredocNode{Identifier: "EOT", Parts: []Node{&Identifier{Name: "foo"}}, Pos: Position{Line: 1, Column: 1}}
	if heredoc.NodeType() != "Heredoc" {
		t.Errorf(errNodeType, heredoc.NodeType())
	}
	if heredoc.GetPos().Line != 1 {
		t.Errorf(errGetPos, heredoc.GetPos())
	}
	heredoc.SetPos(Position{Line: 2, Column: 2})
	if heredoc.GetPos().Line != 2 {
		t.Errorf(errSetPos, heredoc.GetPos())
	}
	if heredoc.String() == "" {
		t.Error(errStringEmpty)
	}
	if heredoc.TokenLiteral() != "EOT" {
		t.Errorf(errTokenLiteral, heredoc.TokenLiteral())
	}
}

func TestTernaryExpr(t *testing.T) {
	ternary := &TernaryExpr{Condition: &Identifier{Name: "cond"}, IfTrue: &Identifier{Name: "yes"}, IfFalse: &Identifier{Name: "no"}, Pos: Position{Line: 1, Column: 1}}
	if ternary.NodeType() != "TernaryExpr" {
		t.Errorf(errNodeType, ternary.NodeType())
	}
	if ternary.GetPos().Line != 1 {
		t.Errorf(errGetPos, ternary.GetPos())
	}
	ternary.SetPos(Position{Line: 2, Column: 2})
	if ternary.GetPos().Line != 2 {
		t.Errorf(errSetPos, ternary.GetPos())
	}
	if ternary.String() == "" {
		t.Error(errStringEmpty)
	}
	if ternary.TokenLiteral() != "?" {
		t.Errorf(errTokenLiteral, ternary.TokenLiteral())
	}
}

func TestPropertyFetchNode(t *testing.T) {
	obj := &Identifier{Name: "obj"}
	prop := "prop"
	pf := &PropertyFetchNode{Object: obj, Property: prop, Pos: Position{Line: 1, Column: 1}}
	if pf.NodeType() != "PropertyFetch" {
		t.Errorf(errNodeType, pf.NodeType())
	}
	if pf.GetPos().Line != 1 {
		t.Errorf(errGetPos, pf.GetPos())
	}
	pf.SetPos(Position{Line: 2, Column: 2})
	if pf.GetPos().Line != 2 {
		t.Errorf(errSetPos, pf.GetPos())
	}
	if pf.String() == "" {
		t.Error(errStringEmpty)
	}
	if pf.TokenLiteral() != prop {
		t.Errorf(errTokenLiteral, pf.TokenLiteral())
	}
}

func TestForeachNode(t *testing.T) {
	expr := &Identifier{Name: "arr"}
	key := &Identifier{Name: "k"}
	val := &Identifier{Name: "v"}
	body := []Node{&Identifier{Name: "stmt"}}
	foreach := &ForeachNode{Expr: expr, KeyVar: key, ValueVar: val, ByRef: false, Body: body, Pos: Position{Line: 1, Column: 1}}
	if foreach.NodeType() != "Foreach" {
		t.Errorf(errNodeType, foreach.NodeType())
	}
	if foreach.GetPos().Line != 1 {
		t.Errorf(errGetPos, foreach.GetPos())
	}
	foreach.SetPos(Position{Line: 2, Column: 2})
	if foreach.GetPos().Line != 2 {
		t.Errorf(errSetPos, foreach.GetPos())
	}
	if foreach.String() == "" {
		t.Error(errStringEmpty)
	}
	if foreach.TokenLiteral() != "foreach" {
		t.Errorf(errTokenLiteral, foreach.TokenLiteral())
	}
}

func TestForeachNodeKeyVarNil(t *testing.T) {
	expr := &Identifier{Name: "arr"}
	val := &Identifier{Name: "v"}
	body := []Node{&Identifier{Name: "stmt"}}
	foreach := &ForeachNode{Expr: expr, KeyVar: nil, ValueVar: val, ByRef: false, Body: body, Pos: Position{Line: 1, Column: 1}}
	str := foreach.String()
	if foreach.NodeType() != "Foreach" {
		t.Errorf(errNodeType, foreach.NodeType())
	}
	if foreach.GetPos().Line != 1 {
		t.Errorf(errGetPos, foreach.GetPos())
	}
	foreach.SetPos(Position{Line: 2, Column: 2})
	if foreach.GetPos().Line != 2 {
		t.Errorf(errSetPos, foreach.GetPos())
	}
	if str == "" {
		t.Error(errStringEmpty)
	}
	if foreach.TokenLiteral() != "foreach" {
		t.Errorf(errTokenLiteral, foreach.TokenLiteral())
	}
	// Ensure the string does not contain "=>" when KeyVar is nil
	if got := str; got != "Foreach(arr as v) @ 2:2" && got != "Foreach(arr as v) @ 1:1" {
		t.Errorf("unexpected String() output: %q", got)
	}
}

func TestThrowNode(t *testing.T) {
	throw := &ThrowNode{Expr: &Identifier{Name: "ex"}, Pos: Position{Line: 1, Column: 1}}
	if throw.NodeType() != "Throw" {
		t.Errorf(errNodeType, throw.NodeType())
	}
	if throw.GetPos().Line != 1 {
		t.Errorf(errGetPos, throw.GetPos())
	}
	throw.SetPos(Position{Line: 2, Column: 2})
	if throw.GetPos().Line != 2 {
		t.Errorf(errSetPos, throw.GetPos())
	}
	if throw.String() == "" {
		t.Error(errStringEmpty)
	}
	if throw.TokenLiteral() != "throw" {
		t.Errorf(errTokenLiteral, throw.TokenLiteral())
	}
}

func TestBooleanLiteralNode(t *testing.T) {
	b := &BooleanLiteral{Value: true, Pos: Position{Line: 1, Column: 2}}
	if b.NodeType() != "BooleanLiteral" {
		t.Errorf(errNodeType, b.NodeType())
	}
	if b.GetPos().Line != 1 || b.GetPos().Column != 2 {
		t.Errorf(errGetPos, b.GetPos())
	}
	b.SetPos(Position{Line: 3, Column: 4})
	if b.GetPos().Line != 3 || b.GetPos().Column != 4 {
		t.Errorf(errSetPos, b.GetPos())
	}
	if b.String() == "" {
		t.Error(errStringEmpty)
	}
	if b.TokenLiteral() != "true" {
		t.Errorf(errTokenLiteral, b.TokenLiteral())
	}
	if val, ok := b.GetValue().(bool); !ok || val != true {
		t.Errorf(errGetValue, b.GetValue())
	}
}

func TestBooleanNodeFalse(t *testing.T) {
	b := &BooleanNode{Value: false, Pos: Position{Line: 1, Column: 1}}
	if b.NodeType() != "Boolean" {
		t.Errorf(errNodeType, b.NodeType())
	}
	if b.GetPos().Line != 1 || b.GetPos().Column != 1 {
		t.Errorf(errGetPos, b.GetPos())
	}
	b.SetPos(Position{Line: 2, Column: 2})
	if b.GetPos().Line != 2 || b.GetPos().Column != 2 {
		t.Errorf(errSetPos, b.GetPos())
	}
	if b.String() == "" {
		t.Error(errStringEmpty)
	}
	if b.TokenLiteral() != "false" {
		t.Errorf(errTokenLiteral, b.TokenLiteral())
	}
}

func TestNullLiteralNode(t *testing.T) {
	n := &NullLiteral{Pos: Position{Line: 2, Column: 3}}
	if n.NodeType() != "NullLiteral" {
		t.Errorf(errNodeType, n.NodeType())
	}
	if n.GetPos().Line != 2 || n.GetPos().Column != 3 {
		t.Errorf(errGetPos, n.GetPos())
	}
	n.SetPos(Position{Line: 4, Column: 5})
	if n.GetPos().Line != 4 || n.GetPos().Column != 5 {
		t.Errorf(errSetPos, n.GetPos())
	}
	if n.String() == "" {
		t.Error(errStringEmpty)
	}
	if n.TokenLiteral() != "null" {
		t.Errorf(errTokenLiteral, n.TokenLiteral())
	}
	if n.GetValue() != nil {
		t.Errorf(errGetValue, n.GetValue())
	}
}

func TestAssignmentNode(t *testing.T) {
	left := &VariableNode{Name: "foo", Pos: Position{Line: 1, Column: 1}}
	right := &IntegerLiteral{Value: 42, Pos: Position{Line: 1, Column: 2}}
	a := &AssignmentNode{Left: left, Operator: "=", Right: right, Pos: Position{Line: 3, Column: 4}}
	if a.NodeType() != "Assignment" {
		t.Errorf(errNodeType, a.NodeType())
	}
	if a.GetPos().Line != 3 || a.GetPos().Column != 4 {
		t.Errorf(errGetPos, a.GetPos())
	}
	a.SetPos(Position{Line: 5, Column: 6})
	if a.GetPos().Line != 5 || a.GetPos().Column != 6 {
		t.Errorf(errSetPos, a.GetPos())
	}
	if a.String() == "" {
		t.Error(errStringEmpty)
	}
	if a.TokenLiteral() != "=" {
		t.Errorf(errTokenLiteral, a.TokenLiteral())
	}
}

func TestReturnNode(t *testing.T) {
	expr := &IntegerLiteral{Value: 123, Pos: Position{Line: 1, Column: 1}}
	r := &ReturnNode{Expr: expr, Pos: Position{Line: 2, Column: 2}}
	if r.NodeType() != "Return" {
		t.Errorf(errNodeType, r.NodeType())
	}
	if r.GetPos().Line != 2 || r.GetPos().Column != 2 {
		t.Errorf(errGetPos, r.GetPos())
	}
	r.SetPos(Position{Line: 3, Column: 3})
	if r.GetPos().Line != 3 || r.GetPos().Column != 3 {
		t.Errorf(errSetPos, r.GetPos())
	}
	if r.String() == "" {
		t.Error(errStringEmpty)
	}
	if r.TokenLiteral() != "return" {
		t.Errorf(errTokenLiteral, r.TokenLiteral())
	}
}

func TestExpressionStmt(t *testing.T) {
	expr := &StringLiteral{Value: "hello", Pos: Position{Line: 1, Column: 1}}
	e := &ExpressionStmt{Expr: expr, Pos: Position{Line: 2, Column: 2}}
	if e.NodeType() != "ExpressionStmt" {
		t.Errorf(errNodeType, e.NodeType())
	}
	if e.GetPos().Line != 2 || e.GetPos().Column != 2 {
		t.Errorf(errGetPos, e.GetPos())
	}
	e.SetPos(Position{Line: 3, Column: 3})
	if e.GetPos().Line != 3 || e.GetPos().Column != 3 {
		t.Errorf(errSetPos, e.GetPos())
	}
	if e.String() == "" {
		t.Error(errStringEmpty)
	}
	if e.TokenLiteral() != "hello" {
		t.Errorf(errTokenLiteral, e.TokenLiteral())
	}
}

func TestBinaryExpr(t *testing.T) {
	left := &IntegerLiteral{Value: 1, Pos: Position{Line: 1, Column: 1}}
	right := &IntegerLiteral{Value: 2, Pos: Position{Line: 1, Column: 2}}
	b := &BinaryExpr{Left: left, Operator: "+", Right: right, Pos: Position{Line: 2, Column: 2}}
	if b.NodeType() != "BinaryExpr" {
		t.Errorf(errNodeType, b.NodeType())
	}
	if b.GetPos().Line != 2 || b.GetPos().Column != 2 {
		t.Errorf(errGetPos, b.GetPos())
	}
	b.SetPos(Position{Line: 3, Column: 3})
	if b.GetPos().Line != 3 || b.GetPos().Column != 3 {
		t.Errorf(errSetPos, b.GetPos())
	}
	if b.String() == "" {
		t.Error(errStringEmpty)
	}
	if b.TokenLiteral() != "+" {
		t.Errorf(errTokenLiteral, b.TokenLiteral())
	}
}

func TestIfNode(t *testing.T) {
	cond := &BooleanLiteral{Value: true, Pos: Position{Line: 1, Column: 1}}
	body := []Node{&ExpressionStmt{Expr: &StringLiteral{Value: "body", Pos: Position{Line: 2, Column: 2}}, Pos: Position{Line: 2, Column: 2}}}
	ifNode := &IfNode{Condition: cond, Body: body, ElseIfs: nil, Else: nil, Pos: Position{Line: 3, Column: 3}}
	if ifNode.NodeType() != "If" {
		t.Errorf(errNodeType, ifNode.NodeType())
	}
	if ifNode.GetPos().Line != 3 || ifNode.GetPos().Column != 3 {
		t.Errorf(errGetPos, ifNode.GetPos())
	}
	ifNode.SetPos(Position{Line: 4, Column: 4})
	if ifNode.GetPos().Line != 4 || ifNode.GetPos().Column != 4 {
		t.Errorf(errSetPos, ifNode.GetPos())
	}
	if ifNode.String() == "" {
		t.Error(errStringEmpty)
	}
	if ifNode.TokenLiteral() != "if" {
		t.Errorf(errTokenLiteral, ifNode.TokenLiteral())
	}
}

func TestElseIfNode(t *testing.T) {
	cond := &BooleanLiteral{Value: false, Pos: Position{Line: 1, Column: 1}}
	body := []Node{&ExpressionStmt{Expr: &StringLiteral{Value: "elseif", Pos: Position{Line: 2, Column: 2}}, Pos: Position{Line: 2, Column: 2}}}
	elseif := &ElseIfNode{Condition: cond, Body: body, Pos: Position{Line: 3, Column: 3}}
	if elseif.NodeType() != "ElseIf" {
		t.Errorf(errNodeType, elseif.NodeType())
	}
	if elseif.GetPos().Line != 3 || elseif.GetPos().Column != 3 {
		t.Errorf(errGetPos, elseif.GetPos())
	}
	elseif.SetPos(Position{Line: 4, Column: 4})
	if elseif.GetPos().Line != 4 || elseif.GetPos().Column != 4 {
		t.Errorf(errSetPos, elseif.GetPos())
	}
	if elseif.String() == "" {
		t.Error(errStringEmpty)
	}
	if elseif.TokenLiteral() != "elseif" {
		t.Errorf(errTokenLiteral, elseif.TokenLiteral())
	}
}

func TestElseNode(t *testing.T) {
	body := []Node{&ExpressionStmt{Expr: &StringLiteral{Value: "else", Pos: Position{Line: 2, Column: 2}}, Pos: Position{Line: 2, Column: 2}}}
	elseNode := &ElseNode{Body: body, Pos: Position{Line: 3, Column: 3}}
	if elseNode.NodeType() != "Else" {
		t.Errorf(errNodeType, elseNode.NodeType())
	}
	if elseNode.GetPos().Line != 3 || elseNode.GetPos().Column != 3 {
		t.Errorf(errGetPos, elseNode.GetPos())
	}
	elseNode.SetPos(Position{Line: 4, Column: 4})
	if elseNode.GetPos().Line != 4 || elseNode.GetPos().Column != 4 {
		t.Errorf(errSetPos, elseNode.GetPos())
	}
	if elseNode.String() == "" {
		t.Error(errStringEmpty)
	}
	if elseNode.TokenLiteral() != "else" {
		t.Errorf(errTokenLiteral, elseNode.TokenLiteral())
	}
}

func TestWhileNode(t *testing.T) {
	cond := &BooleanLiteral{Value: true, Pos: Position{Line: 1, Column: 1}}
	body := []Node{&ExpressionStmt{Expr: &StringLiteral{Value: "while", Pos: Position{Line: 2, Column: 2}}, Pos: Position{Line: 2, Column: 2}}}
	whileNode := &WhileNode{Condition: cond, Body: body, Pos: Position{Line: 3, Column: 3}}
	if whileNode.NodeType() != "While" {
		t.Errorf(errNodeType, whileNode.NodeType())
	}
	if whileNode.GetPos().Line != 3 || whileNode.GetPos().Column != 3 {
		t.Errorf(errGetPos, whileNode.GetPos())
	}
	whileNode.SetPos(Position{Line: 4, Column: 4})
	if whileNode.GetPos().Line != 4 || whileNode.GetPos().Column != 4 {
		t.Errorf(errSetPos, whileNode.GetPos())
	}
	if whileNode.String() == "" {
		t.Error(errStringEmpty)
	}
	if whileNode.TokenLiteral() != "while" {
		t.Errorf(errTokenLiteral, whileNode.TokenLiteral())
	}
}

func TestFunctionDecl(t *testing.T) {
	params := []*Variable{{Name: "a", Pos: Position{Line: 1, Column: 2}}}
	body := []Node{&ExpressionStmt{Expr: &StringLiteral{Value: "body", Pos: Position{Line: 2, Column: 2}}, Pos: Position{Line: 2, Column: 2}}}
	fn := &FunctionDecl{Name: "myFunc", Params: params, Body: body, Pos: Position{Line: 3, Column: 3}}
	if fn.NodeType() != "Function" {
		t.Errorf(errNodeType, fn.NodeType())
	}
	if fn.GetPos().Line != 3 || fn.GetPos().Column != 3 {
		t.Errorf(errGetPos, fn.GetPos())
	}
	fn.SetPos(Position{Line: 4, Column: 4})
	if fn.GetPos().Line != 4 || fn.GetPos().Column != 4 {
		t.Errorf(errSetPos, fn.GetPos())
	}
	if fn.String() == "" {
		t.Error(errStringEmpty)
	}
	if fn.TokenLiteral() != "function" {
		t.Errorf(errTokenLiteral, fn.TokenLiteral())
	}
}

func TestVariable(t *testing.T) {
	v := &Variable{Name: "foo", Pos: Position{Line: 1, Column: 2}}
	if v.NodeType() != "Variable" {
		t.Errorf(errNodeType, v.NodeType())
	}
	if v.GetPos().Line != 1 || v.GetPos().Column != 2 {
		t.Errorf(errGetPos, v.GetPos())
	}
	v.SetPos(Position{Line: 3, Column: 4})
	if v.GetPos().Line != 3 || v.GetPos().Column != 4 {
		t.Errorf(errSetPos, v.GetPos())
	}
	if v.String() == "" {
		t.Error(errStringEmpty)
	}
	if v.TokenLiteral() != "foo" {
		t.Errorf(errTokenLiteral, v.TokenLiteral())
	}
}

func TestFunctionCall(t *testing.T) {
	args := []Node{&IntegerLiteral{Value: 1, Pos: Position{Line: 1, Column: 1}}}
	fc := &FunctionCall{Name: "myFunc", Arguments: args, Pos: Position{Line: 2, Column: 2}}
	if fc.NodeType() != "FunctionCall" {
		t.Errorf(errNodeType, fc.NodeType())
	}
	if fc.GetPos().Line != 2 || fc.GetPos().Column != 2 {
		t.Errorf(errGetPos, fc.GetPos())
	}
	fc.SetPos(Position{Line: 3, Column: 3})
	if fc.GetPos().Line != 3 || fc.GetPos().Column != 3 {
		t.Errorf(errSetPos, fc.GetPos())
	}
	if fc.String() == "" {
		t.Error(errStringEmpty)
	}
	if fc.TokenLiteral() != "myFunc" {
		t.Errorf(errTokenLiteral, fc.TokenLiteral())
	}
}

func TestIdentifierNode2(t *testing.T) {
	id := &IdentifierNode{Value: "foo", Pos: Position{Line: 1, Column: 2}}
	if id.NodeType() != "Identifier" {
		t.Errorf(errNodeType, id.NodeType())
	}
	if id.GetPos().Line != 1 || id.GetPos().Column != 2 {
		t.Errorf(errGetPos, id.GetPos())
	}
	id.SetPos(Position{Line: 3, Column: 4})
	if id.GetPos().Line != 3 || id.GetPos().Column != 4 {
		t.Errorf(errSetPos, id.GetPos())
	}
	if id.String() == "" {
		t.Error(errStringEmpty)
	}
	if id.TokenLiteral() != "foo" {
		t.Errorf(errTokenLiteral, id.TokenLiteral())
	}
}

func TestBooleanNode(t *testing.T) {
	b := &BooleanNode{Value: true, Pos: Position{Line: 1, Column: 1}}
	if b.NodeType() != "Boolean" {
		t.Errorf(errNodeType, b.NodeType())
	}
	if b.GetPos().Line != 1 || b.GetPos().Column != 1 {
		t.Errorf(errGetPos, b.GetPos())
	}
	b.SetPos(Position{Line: 2, Column: 2})
	if b.GetPos().Line != 2 || b.GetPos().Column != 2 {
		t.Errorf(errSetPos, b.GetPos())
	}
	if b.String() == "" {
		t.Error(errStringEmpty)
	}
	if b.TokenLiteral() != "true" {
		t.Errorf(errTokenLiteral, b.TokenLiteral())
	}
}

func TestNullNode(t *testing.T) {
	n := &NullNode{Pos: Position{Line: 1, Column: 1}}
	if n.NodeType() != "Null" {
		t.Errorf(errNodeType, n.NodeType())
	}
	if n.GetPos().Line != 1 || n.GetPos().Column != 1 {
		t.Errorf(errGetPos, n.GetPos())
	}
	n.SetPos(Position{Line: 2, Column: 2})
	if n.GetPos().Line != 2 || n.GetPos().Column != 2 {
		t.Errorf(errSetPos, n.GetPos())
	}
	if n.String() == "" {
		t.Error(errStringEmpty)
	}
	if n.TokenLiteral() != "null" {
		t.Errorf(errTokenLiteral, n.TokenLiteral())
	}
}

func TestConcatNode(t *testing.T) {
	part1 := &StringLiteral{Value: "a", Pos: Position{Line: 1, Column: 1}}
	part2 := &StringLiteral{Value: "b", Pos: Position{Line: 1, Column: 2}}
	c := &ConcatNode{Parts: []Node{part1, part2}, Pos: Position{Line: 2, Column: 2}}
	if c.NodeType() != "Concat" {
		t.Errorf(errNodeType, c.NodeType())
	}
	if c.GetPos().Line != 2 || c.GetPos().Column != 2 {
		t.Errorf(errGetPos, c.GetPos())
	}
	c.SetPos(Position{Line: 3, Column: 3})
	if c.GetPos().Line != 3 || c.GetPos().Column != 3 {
		t.Errorf(errSetPos, c.GetPos())
	}
	if c.String() == "" {
		t.Error(errStringEmpty)
	}
	if c.TokenLiteral() != "." {
		t.Errorf(errTokenLiteral, c.TokenLiteral())
	}
}

func TestAttributeNode(t *testing.T) {
	a := &AttributeNode{Name: "MyAttr", Arguments: nil, Pos: Position{Line: 1, Column: 1}}
	if a.NodeType() != "Attribute" {
		t.Errorf(errNodeType, a.NodeType())
	}
	if a.GetPos().Line != 1 || a.GetPos().Column != 1 {
		t.Errorf(errGetPos, a.GetPos())
	}
	a.SetPos(Position{Line: 2, Column: 2})
	if a.GetPos().Line != 2 || a.GetPos().Column != 2 {
		t.Errorf(errSetPos, a.GetPos())
	}
	if a.String() == "" {
		t.Error(errStringEmpty)
	}
	if a.TokenLiteral() != "MyAttr" {
		t.Errorf(errTokenLiteral, a.TokenLiteral())
	}
}

func TestNamespaceNode(t *testing.T) {
	n := &NamespaceNode{Name: "Foo\\Bar", Body: nil, Pos: Position{Line: 1, Column: 1}}
	if n.NodeType() != "Namespace" {
		t.Errorf(errNodeType, n.NodeType())
	}
	if n.GetPos().Line != 1 || n.GetPos().Column != 1 {
		t.Errorf(errGetPos, n.GetPos())
	}
	n.SetPos(Position{Line: 2, Column: 2})
	if n.GetPos().Line != 2 || n.GetPos().Column != 2 {
		t.Errorf(errSetPos, n.GetPos())
	}
	if n.String() == "" {
		t.Error(errStringEmpty)
	}
	if n.TokenLiteral() != "namespace" {
		t.Errorf(errTokenLiteral, n.TokenLiteral())
	}
}

func TestUseNode(t *testing.T) {
	u := &UseNode{Path: "Foo\\Bar", Alias: "Bar", Type: "class", Pos: Position{Line: 1, Column: 1}}
	if u.NodeType() != "Use" {
		t.Errorf(errNodeType, u.NodeType())
	}
	if u.GetPos().Line != 1 || u.GetPos().Column != 1 {
		t.Errorf(errGetPos, u.GetPos())
	}
	u.SetPos(Position{Line: 2, Column: 2})
	if u.GetPos().Line != 2 || u.GetPos().Column != 2 {
		t.Errorf(errSetPos, u.GetPos())
	}
	if u.String() == "" {
		t.Error(errStringEmpty)
	}
	if u.TokenLiteral() != "use" {
		t.Errorf(errTokenLiteral, u.TokenLiteral())
	}
}

func TestUseNodeNoAlias(t *testing.T) {
	u := &UseNode{Path: "Foo\\Bar", Alias: "", Type: "class", Pos: Position{Line: 1, Column: 1}}
	if u.NodeType() != "Use" {
		t.Errorf(errNodeType, u.NodeType())
	}
	if u.GetPos().Line != 1 || u.GetPos().Column != 1 {
		t.Errorf(errGetPos, u.GetPos())
	}
	u.SetPos(Position{Line: 2, Column: 2})
	if u.GetPos().Line != 2 || u.GetPos().Column != 2 {
		t.Errorf(errSetPos, u.GetPos())
	}
	if u.String() != "use Foo\\Bar @ 2:2" && u.String() != "use Foo\\Bar @ 1:1" {
		t.Errorf("unexpected String() output: %q", u.String())
	}
	if u.TokenLiteral() != "use" {
		t.Errorf(errTokenLiteral, u.TokenLiteral())
	}
}

func TestArrowFunctionNode(t *testing.T) {
	param := &Variable{Name: "x", Pos: Position{Line: 1, Column: 2}}
	expr := &IntegerLiteral{Value: 42, Pos: Position{Line: 2, Column: 3}}
	a := &ArrowFunctionNode{Params: []Node{param}, ReturnType: "int", Expr: expr, Pos: Position{Line: 3, Column: 4}}
	if a.NodeType() != "ArrowFunction" {
		t.Errorf(errNodeType, a.NodeType())
	}
	if a.GetPos().Line != 3 || a.GetPos().Column != 4 {
		t.Errorf(errGetPos, a.GetPos())
	}
	a.SetPos(Position{Line: 5, Column: 6})
	if a.GetPos().Line != 5 || a.GetPos().Column != 6 {
		t.Errorf(errSetPos, a.GetPos())
	}
	if a.String() == "" {
		t.Error(errStringEmpty)
	}
	if a.TokenLiteral() != "fn" {
		t.Errorf(errTokenLiteral, a.TokenLiteral())
	}
}

func TestMatchArmNode(t *testing.T) {
	cond := &IntegerLiteral{Value: 1, Pos: Position{Line: 1, Column: 1}}
	body := &StringLiteral{Value: "one", Pos: Position{Line: 2, Column: 2}}
	arm := &MatchArmNode{Conditions: []Node{cond}, Body: body, Pos: Position{Line: 3, Column: 3}}
	if arm.NodeType() != "MatchArm" {
		t.Errorf(errNodeType, arm.NodeType())
	}
	if arm.GetPos().Line != 3 || arm.GetPos().Column != 3 {
		t.Errorf(errGetPos, arm.GetPos())
	}
	arm.SetPos(Position{Line: 4, Column: 4})
	if arm.GetPos().Line != 4 || arm.GetPos().Column != 4 {
		t.Errorf(errSetPos, arm.GetPos())
	}
	if arm.String() == "" {
		t.Error(errStringEmpty)
	}
	if arm.TokenLiteral() != "=>" {
		t.Errorf(errTokenLiteral, arm.TokenLiteral())
	}
}

func TestMatchNode(t *testing.T) {
	cond := &Variable{Name: "foo", Pos: Position{Line: 1, Column: 1}}
	arm1 := MatchArmNode{Conditions: []Node{&IntegerLiteral{Value: 1}}, Body: &StringLiteral{Value: "one"}, Pos: Position{Line: 2, Column: 2}}
	arm2 := MatchArmNode{Conditions: []Node{&IntegerLiteral{Value: 2}}, Body: &StringLiteral{Value: "two"}, Pos: Position{Line: 3, Column: 3}}
	match := &MatchNode{Condition: cond, Arms: []MatchArmNode{arm1, arm2}, Pos: Position{Line: 4, Column: 4}}
	if match.NodeType() != "Match" {
		t.Errorf(errNodeType, match.NodeType())
	}
	if match.GetPos().Line != 4 || match.GetPos().Column != 4 {
		t.Errorf(errGetPos, match.GetPos())
	}
	match.SetPos(Position{Line: 5, Column: 5})
	if match.GetPos().Line != 5 || match.GetPos().Column != 5 {
		t.Errorf(errSetPos, match.GetPos())
	}
	if match.String() == "" {
		t.Error(errStringEmpty)
	}
	if match.TokenLiteral() != "match" {
		t.Errorf(errTokenLiteral, match.TokenLiteral())
	}
}
