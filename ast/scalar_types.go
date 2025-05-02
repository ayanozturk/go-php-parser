package ast

import (
	"fmt"
	"strconv"
)

// StringNode represents a PHP string literal
type StringNode struct {
	Value string
	Pos   Position
}

func (s *StringNode) NodeType() string    { return "String" }
func (s *StringNode) GetPos() Position    { return s.Pos }
func (s *StringNode) SetPos(pos Position) { s.Pos = pos }
func (s *StringNode) String() string {
	return fmt.Sprintf("\"%s\" @ %d:%d", s.Value, s.Pos.Line, s.Pos.Column)
}
func (s *StringNode) TokenLiteral() string {
	return strconv.Quote(s.Value)
}

// IntegerNode represents an integer literal
type IntegerNode struct {
	Value int64
	Pos   Position
}

func (i *IntegerNode) NodeType() string    { return "Integer" }
func (i *IntegerNode) GetPos() Position    { return i.Pos }
func (i *IntegerNode) SetPos(pos Position) { i.Pos = pos }
func (i *IntegerNode) String() string {
	return fmt.Sprintf("%d @ %d:%d", i.Value, i.Pos.Line, i.Pos.Column)
}
func (i *IntegerNode) TokenLiteral() string {
	return strconv.FormatInt(i.Value, 10)
}

// FloatNode represents a float literal
type FloatNode struct {
	Value float64
	Pos   Position
}

func (f *FloatNode) NodeType() string    { return "Float" }
func (f *FloatNode) GetPos() Position    { return f.Pos }
func (f *FloatNode) SetPos(pos Position) { f.Pos = pos }
func (f *FloatNode) String() string {
	return fmt.Sprintf("%f @ %d:%d", f.Value, f.Pos.Line, f.Pos.Column)
}
func (f *FloatNode) TokenLiteral() string {
	return strconv.FormatFloat(f.Value, 'f', -1, 64)
}

// ScalarType represents PHP scalar types (int, float, string, bool)
type ScalarType string

const (
	ScalarInt    ScalarType = "int"
	ScalarFloat  ScalarType = "float"
	ScalarString ScalarType = "string"
	ScalarBool   ScalarType = "bool"
)

func IsScalarType(t string) bool {
	switch ScalarType(t) {
	case ScalarInt, ScalarFloat, ScalarString, ScalarBool:
		return true
	default:
		return false
	}
}
