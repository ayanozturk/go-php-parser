package ast

import "testing"

func TestArrayNode(t *testing.T) {
	elem := &StringLiteral{Value: "foo", Pos: Position{Line: 1, Column: 1}}
	a := &ArrayNode{Elements: []Node{elem}, Pos: Position{Line: 2, Column: 2}}
	if a.NodeType() != "Array" {
		t.Errorf(errNodeType, a.NodeType())
	}
	if a.GetPos().Line != 2 || a.GetPos().Column != 2 {
		t.Errorf(errGetPos, a.GetPos())
	}
	a.SetPos(Position{Line: 3, Column: 3})
	if a.GetPos().Line != 3 || a.GetPos().Column != 3 {
		t.Errorf(errSetPos, a.GetPos())
	}
	if a.String() == "" {
		t.Error(errStringEmpty)
	}
	if a.TokenLiteral() != "array" {
		t.Errorf(errTokenLiteral, a.TokenLiteral())
	}
	if len(a.Elements) != 1 || a.Elements[0] != elem {
		t.Errorf("Elements: got %+v", a.Elements)
	}
}

func TestKeyValueNode(t *testing.T) {
	key := &StringLiteral{Value: "k", Pos: Position{Line: 1, Column: 1}}
	val := &StringLiteral{Value: "v", Pos: Position{Line: 1, Column: 2}}
	kv := &KeyValueNode{Key: key, Value: val, Pos: Position{Line: 2, Column: 2}}
	if kv.NodeType() != "KeyValue" {
		t.Errorf(errNodeType, kv.NodeType())
	}
	if kv.GetPos().Line != 2 || kv.GetPos().Column != 2 {
		t.Errorf(errGetPos, kv.GetPos())
	}
	kv.SetPos(Position{Line: 3, Column: 3})
	if kv.GetPos().Line != 3 || kv.GetPos().Column != 3 {
		t.Errorf(errSetPos, kv.GetPos())
	}
	if kv.String() == "" {
		t.Error(errStringEmpty)
	}
	if kv.TokenLiteral() != "=>" {
		t.Errorf(errTokenLiteral, kv.TokenLiteral())
	}
}

func TestKeyValueNodeValueOnlyString(t *testing.T) {
	val := &StringLiteral{Value: "valueOnly", Pos: Position{Line: 1, Column: 2}}
	kv := &KeyValueNode{Key: nil, Value: val, Pos: Position{Line: 2, Column: 2}}
	if got := kv.String(); got != val.String() {
		t.Errorf("KeyValueNode.String() with nil Key: got %q, want %q", got, val.String())
	}
}

func TestKeyValueNodeStringValueOnly(t *testing.T) {
	val := &StringLiteral{Value: "valueOnly", Pos: Position{Line: 1, Column: 2}}
	kv := &KeyValueNode{Key: nil, Value: val, Pos: Position{Line: 2, Column: 2}}
	// Check that TokenLiteral returns the expected value for value-only KeyValueNode
	if got := kv.TokenLiteral(); got != "=>" && got != "" {
		t.Errorf("KeyValueNode.TokenLiteral() with nil Key: got %q, want \"=>\" or \"\"", got)
	}
}

func TestArrayAccessNode(t *testing.T) {
	v := &VariableNode{Name: "arr", Pos: Position{Line: 1, Column: 1}}
	idx := &IntegerLiteral{Value: 0, Pos: Position{Line: 1, Column: 2}}
	aa := &ArrayAccessNode{Var: v, Index: idx, Pos: Position{Line: 2, Column: 2}}
	if aa.NodeType() != "ArrayAccess" {
		t.Errorf(errNodeType, aa.NodeType())
	}
	if aa.GetPos().Line != 2 || aa.GetPos().Column != 2 {
		t.Errorf(errGetPos, aa.GetPos())
	}
	aa.SetPos(Position{Line: 3, Column: 3})
	if aa.GetPos().Line != 3 || aa.GetPos().Column != 3 {
		t.Errorf(errSetPos, aa.GetPos())
	}
	if aa.String() == "" {
		t.Error(errStringEmpty)
	}
	if aa.TokenLiteral() != "[" {
		t.Errorf(errTokenLiteral, aa.TokenLiteral())
	}
}

func TestArrayItemNode(t *testing.T) {
	key := &StringLiteral{Value: "k", Pos: Position{Line: 1, Column: 1}}
	val := &StringLiteral{Value: "v", Pos: Position{Line: 1, Column: 2}}
	item := &ArrayItemNode{Key: key, Value: val, ByRef: true, Unpack: true, Pos: Position{Line: 2, Column: 2}}
	if item.NodeType() != "ArrayItem" {
		t.Errorf(errNodeType, item.NodeType())
	}
	if item.GetPos().Line != 2 || item.GetPos().Column != 2 {
		t.Errorf(errGetPos, item.GetPos())
	}
	item.SetPos(Position{Line: 3, Column: 3})
	if item.GetPos().Line != 3 || item.GetPos().Column != 3 {
		t.Errorf(errSetPos, item.GetPos())
	}
	if item.String() == "" {
		t.Error(errStringEmpty)
	}
	if item.TokenLiteral() != "=>" {
		t.Errorf(errTokenLiteral, item.TokenLiteral())
	}
	item2 := &ArrayItemNode{Key: nil, Value: val, ByRef: false, Unpack: false, Pos: Position{Line: 4, Column: 4}}
	if item2.TokenLiteral() != "" {
		t.Errorf("TokenLiteral: expected empty, got %q", item2.TokenLiteral())
	}
}

func TestArrayItemNodeStringValueOnly(t *testing.T) {
	val := &StringLiteral{Value: "foo", Pos: Position{Line: 3, Column: 4}}
	item := &ArrayItemNode{Key: nil, Value: val, ByRef: false, Unpack: false, Pos: Position{Line: 4, Column: 5}}
	expected := "ArrayItem(foo) @ 4:5"
	if got := item.String(); got != expected {
		t.Errorf("ArrayItemNode.String() with nil Key: got %q, want %q", got, expected)
	}
}

func TestArrayItemNodeStringValueOnlyByRefUnpack(t *testing.T) {
	val := &StringLiteral{Value: "foo", Pos: Position{Line: 3, Column: 4}}
	item := &ArrayItemNode{Key: nil, Value: val, ByRef: true, Unpack: true, Pos: Position{Line: 4, Column: 5}}
	expected := "ArrayItem(&...foo) @ 4:5"
	if got := item.String(); got != expected {
		t.Errorf("ArrayItemNode.String() with ByRef/Unpack: got %q, want %q", got, expected)
	}
}
