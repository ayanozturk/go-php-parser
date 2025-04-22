package ast

import (
	"strings"
	"testing"
)

func TestFunctionNodeMethods(t *testing.T) {
	param := &Identifier{Name: "arg1", Pos: Position{Line: 2, Column: 3}}
	bodyStmt := &StringLiteral{Value: "body", Pos: Position{Line: 4, Column: 5}}
	fn := &FunctionNode{
		Name:       "myFunc",
		Visibility: "public",
		Modifiers:  []string{"static"},
		ReturnType: "int",
		Params:     []Node{param},
		Body:       []Node{bodyStmt},
		Pos:        Position{Line: 1, Column: 1},
	}
	if fn.NodeType() != "Function" {
		t.Errorf("NodeType: got %q", fn.NodeType())
	}
	if fn.GetPos().Line != 1 || fn.GetPos().Column != 1 {
		t.Errorf("GetPos: got %+v", fn.GetPos())
	}
	fn.SetPos(Position{Line: 5, Column: 6})
	if fn.GetPos().Line != 5 || fn.GetPos().Column != 6 {
		t.Errorf("SetPos: got %+v", fn.GetPos())
	}
	str := fn.String()
	if str == "" || str == "Function(myFunc) : int @ 5:6" {
		t.Errorf("String: got %q", str)
	}
	if fn.TokenLiteral() != "function" {
		t.Errorf("TokenLiteral: got %q", fn.TokenLiteral())
	}
	if len(fn.Params) != 1 || fn.Params[0] != param {
		t.Errorf("Params: got %+v", fn.Params)
	}
	if len(fn.Body) != 1 || fn.Body[0] != bodyStmt {
		t.Errorf("Body: got %+v", fn.Body)
	}
}

func TestFunctionCallNodeMethods(t *testing.T) {
	name := &Identifier{Name: "foo", Pos: Position{Line: 2, Column: 3}}
	arg1 := &StringLiteral{Value: "bar", Pos: Position{Line: 4, Column: 5}}
	arg2 := &IntegerLiteral{Value: 42, Pos: Position{Line: 6, Column: 7}}
	call := &FunctionCallNode{
		Name: name,
		Args: []Node{arg1, arg2},
		Pos:  Position{Line: 1, Column: 1},
	}
	if call.NodeType() != "FunctionCall" {
		t.Errorf("NodeType: got %q", call.NodeType())
	}
	if call.GetPos().Line != 1 || call.GetPos().Column != 1 {
		t.Errorf("GetPos: got %+v", call.GetPos())
	}
	call.SetPos(Position{Line: 8, Column: 9})
	if call.GetPos().Line != 8 || call.GetPos().Column != 9 {
		t.Errorf("SetPos: got %+v", call.GetPos())
	}
	str := call.String()
	if str == "" {
		t.Errorf("String: got %q", str)
	}
	if call.TokenLiteral() != "foo" {
		t.Errorf("TokenLiteral: got %q", call.TokenLiteral())
	}
	if len(call.Args) != 2 || call.Args[0] != arg1 || call.Args[1] != arg2 {
		t.Errorf("Args: got %+v", call.Args)
	}
}

func TestUnpackedArgumentNodeMethods(t *testing.T) {
	expr := &Identifier{Name: "baz", Pos: Position{Line: 2, Column: 3}}
	u := &UnpackedArgumentNode{
		Expr: expr,
		Pos:  Position{Line: 1, Column: 1},
	}
	if u.NodeType() != "UnpackedArgument" {
		t.Errorf("NodeType: got %q", u.NodeType())
	}
	if u.GetPos().Line != 1 || u.GetPos().Column != 1 {
		t.Errorf("GetPos: got %+v", u.GetPos())
	}
	u.SetPos(Position{Line: 4, Column: 5})
	if u.GetPos().Line != 4 || u.GetPos().Column != 5 {
		t.Errorf("SetPos: got %+v", u.GetPos())
	}
	str := u.String()
	if str == "" || str == "...baz" {
		t.Errorf("String: got %q", str)
	}
	if u.TokenLiteral() != "..." {
		t.Errorf("TokenLiteral: got %q", u.TokenLiteral())
	}
}

func TestStaticFunctionNode(t *testing.T) {
	param := &Identifier{Name: "x", Pos: Position{Line: 2, Column: 1}}
	fn := &FunctionNode{
		Name:       "staticFunc",
		Visibility: "public",
		Modifiers:  []string{"static"},
		ReturnType: "void",
		Params:     []Node{param},
		Body:       nil,
		Pos:        Position{Line: 1, Column: 1},
	}
	if len(fn.Modifiers) == 0 || fn.Modifiers[0] != "static" {
		t.Errorf("Expected 'static' modifier, got %+v", fn.Modifiers)
	}
	if !strings.Contains(fn.String(), "static") {
		t.Errorf("String() should include 'static', got %q", fn.String())
	}
}
