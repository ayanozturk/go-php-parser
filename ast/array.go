package ast

import "fmt"

// ArrayNode represents an array literal
type ArrayNode struct {
	Elements []Node
	Pos      Position
}

func (a *ArrayNode) NodeType() string    { return "Array" }
func (a *ArrayNode) GetPos() Position    { return a.Pos }
func (a *ArrayNode) SetPos(pos Position) { a.Pos = pos }
func (a *ArrayNode) String() string {
	return fmt.Sprintf("Array @ %d:%d", a.Pos.Line, a.Pos.Column)
}
func (a *ArrayNode) TokenLiteral() string {
	return "array"
}

// KeyValueNode represents a key-value pair in an array
type KeyValueNode struct {
	Key   Node
	Value Node
	Pos   Position
}

func (kv *KeyValueNode) NodeType() string    { return "KeyValue" }
func (kv *KeyValueNode) GetPos() Position    { return kv.Pos }
func (kv *KeyValueNode) SetPos(pos Position) { kv.Pos = pos }
func (kv *KeyValueNode) String() string {
	if kv.Key == nil {
		return kv.Value.String()
	}
	return kv.Key.String() + " => " + kv.Value.String()
}
func (kv *KeyValueNode) TokenLiteral() string {
	return "=>"
}

// ArrayItemNode represents an item in an array
type ArrayItemNode struct {
	Key    Node // Optional key for associative arrays
	Value  Node // The value of the array item
	ByRef  bool // Whether the value is passed by reference
	Unpack bool // Whether this is a spread operator item (...$array)
	Pos    Position
}

// ArrayAccessNode represents array access expressions like $config['toolbar']
type ArrayAccessNode struct {
	Var   Node // The array variable being accessed
	Index Node // The index/key being accessed
	Pos   Position
}

func (a *ArrayAccessNode) NodeType() string    { return "ArrayAccess" }
func (a *ArrayAccessNode) GetPos() Position    { return a.Pos }
func (a *ArrayAccessNode) SetPos(pos Position) { a.Pos = pos }
func (a *ArrayAccessNode) String() string {
	return fmt.Sprintf("ArrayAccess(%s[%s]) @ %d:%d", a.Var.String(), a.Index.String(), a.Pos.Line, a.Pos.Column)
}
func (a *ArrayAccessNode) TokenLiteral() string { return "[" }

func (a *ArrayItemNode) NodeType() string    { return "ArrayItem" }
func (a *ArrayItemNode) GetPos() Position    { return a.Pos }
func (a *ArrayItemNode) SetPos(pos Position) { a.Pos = pos }
func (a *ArrayItemNode) String() string {
	var prefix string
	if a.ByRef {
		prefix += "&"
	}
	if a.Unpack {
		prefix += "..."
	}
	if a.Key != nil {
		return fmt.Sprintf("ArrayItem(%s%s => %s) @ %d:%d", prefix, a.Key.TokenLiteral(), a.Value.TokenLiteral(), a.Pos.Line, a.Pos.Column)
	}
	return fmt.Sprintf("ArrayItem(%s%s) @ %d:%d", prefix, a.Value.TokenLiteral(), a.Pos.Line, a.Pos.Column)
}
func (a *ArrayItemNode) TokenLiteral() string {
	if a.Key != nil {
		return "=>"
	}
	return ""
}
