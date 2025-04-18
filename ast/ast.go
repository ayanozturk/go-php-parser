package ast

import (
	"fmt"
	"strconv"
	"strings"
)

// Position holds line/column/offset info for tooling
type Position struct {
	Line   int
	Column int
	Offset int
}

// Node is the interface that all AST nodes implement
// NodeType returns a string identifier for the node type
type Node interface {
	NodeType() string
	GetPos() Position
	SetPos(Position)
	String() string
	TokenLiteral() string
}

// Identifier represents a name like variable or function name
type Identifier struct {
	Name string
	Pos  Position
}

func (i *Identifier) NodeType() string    { return "Identifier" }
func (i *Identifier) GetPos() Position    { return i.Pos }
func (i *Identifier) SetPos(pos Position) { i.Pos = pos }
func (i *Identifier) String() string {
	return fmt.Sprintf("Identifier(%s) @ %d:%d", i.Name, i.Pos.Line, i.Pos.Column)
}
func (i *Identifier) TokenLiteral() string {
	return i.Name
}

// VariableNode represents a PHP variable (e.g., $var)
type VariableNode struct {
	Name string // Without the leading $
	Pos  Position
}

func (v *VariableNode) NodeType() string    { return "Variable" }
func (v *VariableNode) GetPos() Position    { return v.Pos }
func (v *VariableNode) SetPos(pos Position) { v.Pos = pos }
func (v *VariableNode) String() string {
	return fmt.Sprintf("Variable($%s) @ %d:%d", v.Name, v.Pos.Line, v.Pos.Column)
}
func (v *VariableNode) TokenLiteral() string {
	return v.Name
}

// LiteralNode represents a literal value - this is now an interface
type LiteralNode interface {
	Node
	GetValue() interface{}
}

// StringLiteral represents a string literal
type StringLiteral struct {
	Value string
	Pos   Position
}

func (s *StringLiteral) NodeType() string    { return "StringLiteral" }
func (s *StringLiteral) GetPos() Position    { return s.Pos }
func (s *StringLiteral) SetPos(pos Position) { s.Pos = pos }
func (s *StringLiteral) String() string {
	return fmt.Sprintf("String(%q) @ %d:%d", s.Value, s.Pos.Line, s.Pos.Column)
}
func (s *StringLiteral) TokenLiteral() string  { return s.Value }
func (s *StringLiteral) GetValue() interface{} { return s.Value }

// InterpolatedStringLiteral represents a string with interpolated expressions
type InterpolatedStringLiteral struct {
	Parts []Node
	Pos   Position
}

func (s *InterpolatedStringLiteral) NodeType() string    { return "InterpolatedString" }
func (s *InterpolatedStringLiteral) GetPos() Position    { return s.Pos }
func (s *InterpolatedStringLiteral) SetPos(pos Position) { s.Pos = pos }
func (s *InterpolatedStringLiteral) String() string {
	return fmt.Sprintf("InterpolatedString @ %d:%d", s.Pos.Line, s.Pos.Column)
}
func (s *InterpolatedStringLiteral) TokenLiteral() string {
	parts := make([]string, len(s.Parts))
	for i, part := range s.Parts {
		parts[i] = part.TokenLiteral()
	}
	return strings.Join(parts, "")
}

// IntegerLiteral represents an integer literal
type IntegerLiteral struct {
	Value int64
	Pos   Position
}

func (i *IntegerLiteral) NodeType() string    { return "IntegerLiteral" }
func (i *IntegerLiteral) GetPos() Position    { return i.Pos }
func (i *IntegerLiteral) SetPos(pos Position) { i.Pos = pos }
func (i *IntegerLiteral) String() string {
	return fmt.Sprintf("Integer(%d) @ %d:%d", i.Value, i.Pos.Line, i.Pos.Column)
}
func (i *IntegerLiteral) TokenLiteral() string  { return fmt.Sprintf("%d", i.Value) }
func (i *IntegerLiteral) GetValue() interface{} { return i.Value }

// FloatLiteral represents a floating point literal
type FloatLiteral struct {
	Value float64
	Pos   Position
}

func (f *FloatLiteral) NodeType() string    { return "FloatLiteral" }
func (f *FloatLiteral) GetPos() Position    { return f.Pos }
func (f *FloatLiteral) SetPos(pos Position) { f.Pos = pos }
func (f *FloatLiteral) String() string {
	return fmt.Sprintf("Float(%g) @ %d:%d", f.Value, f.Pos.Line, f.Pos.Column)
}
func (f *FloatLiteral) TokenLiteral() string  { return fmt.Sprintf("%g", f.Value) }
func (f *FloatLiteral) GetValue() interface{} { return f.Value }

// BooleanLiteral represents a boolean literal
type BooleanLiteral struct {
	Value bool
	Pos   Position
}

func (b *BooleanLiteral) NodeType() string    { return "BooleanLiteral" }
func (b *BooleanLiteral) GetPos() Position    { return b.Pos }
func (b *BooleanLiteral) SetPos(pos Position) { b.Pos = pos }
func (b *BooleanLiteral) String() string {
	return fmt.Sprintf("Boolean(%t) @ %d:%d", b.Value, b.Pos.Line, b.Pos.Column)
}
func (b *BooleanLiteral) TokenLiteral() string  { return fmt.Sprintf("%t", b.Value) }
func (b *BooleanLiteral) GetValue() interface{} { return b.Value }

// NullLiteral represents a null literal
type NullLiteral struct {
	Pos Position
}

func (n *NullLiteral) NodeType() string    { return "NullLiteral" }
func (n *NullLiteral) GetPos() Position    { return n.Pos }
func (n *NullLiteral) SetPos(pos Position) { n.Pos = pos }
func (n *NullLiteral) String() string {
	return fmt.Sprintf("Null @ %d:%d", n.Pos.Line, n.Pos.Column)
}
func (n *NullLiteral) TokenLiteral() string  { return "null" }
func (n *NullLiteral) GetValue() interface{} { return nil }

// AssignmentNode represents a variable assignment
type AssignmentNode struct {
	Left  Node
	Right Node
	Pos   Position
}

func (a *AssignmentNode) NodeType() string    { return "Assignment" }
func (a *AssignmentNode) GetPos() Position    { return a.Pos }
func (a *AssignmentNode) SetPos(pos Position) { a.Pos = pos }
func (a *AssignmentNode) String() string {
	return fmt.Sprintf("Assignment(%s = %s) @ %d:%d", a.Left.String(), a.Right.String(), a.Pos.Line, a.Pos.Column)
}
func (a *AssignmentNode) TokenLiteral() string {
	return "="
}

// ReturnNode represents a return statement
type ReturnNode struct {
	Expr Node
	Pos  Position
}

func (r *ReturnNode) NodeType() string    { return "Return" }
func (r *ReturnNode) GetPos() Position    { return r.Pos }
func (r *ReturnNode) SetPos(pos Position) { r.Pos = pos }
func (r *ReturnNode) String() string {
	return fmt.Sprintf("Return(%s) @ %d:%d", r.Expr.String(), r.Pos.Line, r.Pos.Column)
}
func (r *ReturnNode) TokenLiteral() string {
	return "return"
}

// ExpressionStmt wraps a single expression as a statement
type ExpressionStmt struct {
	Expr Node
	Pos  Position
}

func (e *ExpressionStmt) NodeType() string    { return "ExpressionStmt" }
func (e *ExpressionStmt) GetPos() Position    { return e.Pos }
func (e *ExpressionStmt) SetPos(pos Position) { e.Pos = pos }
func (e *ExpressionStmt) String() string {
	return fmt.Sprintf("ExpressionStmt(%s) @ %d:%d", e.Expr.String(), e.Pos.Line, e.Pos.Column)
}
func (e *ExpressionStmt) TokenLiteral() string {
	return e.Expr.TokenLiteral()
}

// BinaryExpr represents a binary operation
type BinaryExpr struct {
	Left     Node
	Operator string
	Right    Node
	Pos      Position
}

func (b *BinaryExpr) NodeType() string    { return "BinaryExpr" }
func (b *BinaryExpr) GetPos() Position    { return b.Pos }
func (b *BinaryExpr) SetPos(pos Position) { b.Pos = pos }
func (b *BinaryExpr) String() string {
	return fmt.Sprintf("BinaryExpr(%s %s %s) @ %d:%d", b.Left.String(), b.Operator, b.Right.String(), b.Pos.Line, b.Pos.Column)
}
func (b *BinaryExpr) TokenLiteral() string {
	return b.Operator
}

// IfNode represents an if statement with optional elseif/else clauses
type IfNode struct {
	Condition Node
	Body      []Node
	ElseIfs   []*ElseIfNode
	Else      *ElseNode
	Pos       Position
}

func (i *IfNode) NodeType() string    { return "If" }
func (i *IfNode) GetPos() Position    { return i.Pos }
func (i *IfNode) SetPos(pos Position) { i.Pos = pos }
func (i *IfNode) String() string {
	return fmt.Sprintf("If(Cond: %s) @ %d:%d", i.Condition.String(), i.Pos.Line, i.Pos.Column)
}
func (i *IfNode) TokenLiteral() string {
	return "if"
}

// ElseIfNode represents an elseif clause
type ElseIfNode struct {
	Condition Node
	Body      []Node
	Pos       Position
}

func (ei *ElseIfNode) NodeType() string    { return "ElseIf" }
func (ei *ElseIfNode) GetPos() Position    { return ei.Pos }
func (ei *ElseIfNode) SetPos(pos Position) { ei.Pos = pos }
func (ei *ElseIfNode) String() string {
	return fmt.Sprintf("ElseIf(Cond: %s) @ %d:%d", ei.Condition.String(), ei.Pos.Line, ei.Pos.Column)
}
func (ei *ElseIfNode) TokenLiteral() string {
	return "elseif"
}

// ElseNode represents an else clause
type ElseNode struct {
	Body []Node
	Pos  Position
}

func (e *ElseNode) NodeType() string    { return "Else" }
func (e *ElseNode) GetPos() Position    { return e.Pos }
func (e *ElseNode) SetPos(pos Position) { e.Pos = pos }
func (e *ElseNode) String() string {
	return fmt.Sprintf("Else @ %d:%d", e.Pos.Line, e.Pos.Column)
}
func (e *ElseNode) TokenLiteral() string {
	return "else"
}

// WhileNode represents a while loop
type WhileNode struct {
	Condition Node
	Body      []Node
	Pos       Position
}

func (w *WhileNode) NodeType() string    { return "While" }
func (w *WhileNode) GetPos() Position    { return w.Pos }
func (w *WhileNode) SetPos(pos Position) { w.Pos = pos }
func (w *WhileNode) String() string {
	return fmt.Sprintf("While(Cond: %s) @ %d:%d", w.Condition.String(), w.Pos.Line, w.Pos.Column)
}
func (w *WhileNode) TokenLiteral() string {
	return "while"
}

type FunctionDecl struct {
	Name   string
	Params []*Variable
	Body   []Node
	Pos    Position
}

func (fd *FunctionDecl) NodeType() string    { return "Function" }
func (fd *FunctionDecl) GetPos() Position    { return fd.Pos }
func (fd *FunctionDecl) SetPos(pos Position) { fd.Pos = pos }
func (fd *FunctionDecl) String() string {
	return fmt.Sprintf("Function(%s) @ %d:%d", fd.Name, fd.Pos.Line, fd.Pos.Column)
}
func (fd *FunctionDecl) TokenLiteral() string {
	return "function"
}

type Variable struct {
	Name string
	Pos  Position
}

func (v *Variable) NodeType() string    { return "Variable" }
func (v *Variable) GetPos() Position    { return v.Pos }
func (v *Variable) SetPos(pos Position) { v.Pos = pos }
func (v *Variable) String() string {
	return fmt.Sprintf("Variable(%s) @ %d:%d", v.Name, v.Pos.Line, v.Pos.Column)
}
func (v *Variable) TokenLiteral() string {
	return v.Name
}

// Add these new node types
type FunctionCall struct {
	Name      string
	Arguments []Node
	Pos       Position
}

func (f *FunctionCall) NodeType() string    { return "FunctionCall" }
func (f *FunctionCall) GetPos() Position    { return f.Pos }
func (f *FunctionCall) SetPos(pos Position) { f.Pos = pos }
func (f *FunctionCall) String() string {
	return fmt.Sprintf("FunctionCall(%s) @ %d:%d", f.Name, f.Pos.Line, f.Pos.Column)
}
func (f *FunctionCall) TokenLiteral() string {
	return f.Name
}

// ClassNode represents a PHP class definition
type ClassNode struct {
	Name       string
	Extends    string
	Implements []string
	Properties []Node
	Methods    []Node
	Pos        Position
}

// NewNode represents object instantiation
type NewNode struct {
	ClassName string
	Args      []Node
	Pos       Position
}

func (n *NewNode) NodeType() string    { return "New" }
func (n *NewNode) GetPos() Position    { return n.Pos }
func (n *NewNode) SetPos(pos Position) { n.Pos = pos }
func (n *NewNode) String() string {
	return fmt.Sprintf("New(%s) @ %d:%d", n.ClassName, n.Pos.Line, n.Pos.Column)
}
func (n *NewNode) TokenLiteral() string {
	return "new"
}

// MethodCallNode represents a method call on an object
type MethodCallNode struct {
	Object Node
	Method string
	Args   []Node
	Pos    Position
}

func (m *MethodCallNode) NodeType() string    { return "MethodCall" }
func (m *MethodCallNode) GetPos() Position    { return m.Pos }
func (m *MethodCallNode) SetPos(pos Position) { m.Pos = pos }
func (m *MethodCallNode) String() string {
	return fmt.Sprintf("MethodCall(%s) @ %d:%d", m.Method, m.Pos.Line, m.Pos.Column)
}
func (m *MethodCallNode) TokenLiteral() string {
	return m.Method
}

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

// IdentifierNode represents an identifier
type IdentifierNode struct {
	Value string
	Pos   Position
}

func (i *IdentifierNode) NodeType() string    { return "Identifier" }
func (i *IdentifierNode) GetPos() Position    { return i.Pos }
func (i *IdentifierNode) SetPos(pos Position) { i.Pos = pos }
func (i *IdentifierNode) String() string {
	return fmt.Sprintf("%s @ %d:%d", i.Value, i.Pos.Line, i.Pos.Column)
}
func (i *IdentifierNode) TokenLiteral() string {
	return i.Value
}

// StringNode represents a string literal
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
	return s.Value
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

// BooleanNode represents a boolean literal
type BooleanNode struct {
	Value bool
	Pos   Position
}

func (b *BooleanNode) NodeType() string    { return "Boolean" }
func (b *BooleanNode) GetPos() Position    { return b.Pos }
func (b *BooleanNode) SetPos(pos Position) { b.Pos = pos }
func (b *BooleanNode) String() string {
	return fmt.Sprintf("%t @ %d:%d", b.Value, b.Pos.Line, b.Pos.Column)
}
func (b *BooleanNode) TokenLiteral() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// NullNode represents a null literal
type NullNode struct {
	Pos Position
}

func (n *NullNode) NodeType() string    { return "Null" }
func (n *NullNode) GetPos() Position    { return n.Pos }
func (n *NullNode) SetPos(pos Position) { n.Pos = pos }
func (n *NullNode) String() string {
	return fmt.Sprintf("null @ %d:%d", n.Pos.Line, n.Pos.Column)
}
func (n *NullNode) TokenLiteral() string {
	return "null"
}

// ConcatNode represents string concatenation with variable interpolation
type ConcatNode struct {
	Parts []Node
	Pos   Position
}

func (c *ConcatNode) NodeType() string    { return "Concat" }
func (c *ConcatNode) GetPos() Position    { return c.Pos }
func (c *ConcatNode) SetPos(pos Position) { c.Pos = pos }
func (c *ConcatNode) String() string {
	var parts []string
	for _, part := range c.Parts {
		parts = append(parts, part.String())
	}
	return fmt.Sprintf("Concat(%s) @ %d:%d", strings.Join(parts, " . "), c.Pos.Line, c.Pos.Column)
}
func (c *ConcatNode) TokenLiteral() string {
	return "."
}

// ArrayItemNode represents an item in an array
type ArrayItemNode struct {
	Key    Node // Optional key for associative arrays
	Value  Node // The value of the array item
	ByRef  bool // Whether the value is passed by reference
	Unpack bool // Whether this is a spread operator item (...$array)
	Pos    Position
}

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

// AttributeNode represents a PHP 8.0+ attribute
type AttributeNode struct {
	Name      string
	Arguments []Node
	Pos       Position
}

func (a *AttributeNode) NodeType() string    { return "Attribute" }
func (a *AttributeNode) GetPos() Position    { return a.Pos }
func (a *AttributeNode) SetPos(pos Position) { a.Pos = pos }
func (a *AttributeNode) String() string {
	return fmt.Sprintf("#[%s] @ %d:%d", a.Name, a.Pos.Line, a.Pos.Column)
}
func (a *AttributeNode) TokenLiteral() string { return a.Name }

// NamespaceNode represents a PHP namespace declaration
type NamespaceNode struct {
	Name string
	Body []Node
	Pos  Position
}

func (n *NamespaceNode) NodeType() string    { return "Namespace" }
func (n *NamespaceNode) GetPos() Position    { return n.Pos }
func (n *NamespaceNode) SetPos(pos Position) { n.Pos = pos }
func (n *NamespaceNode) String() string {
	return fmt.Sprintf("namespace %s @ %d:%d", n.Name, n.Pos.Line, n.Pos.Column)
}
func (n *NamespaceNode) TokenLiteral() string { return "namespace" }

// UseNode represents a PHP use statement
type UseNode struct {
	Path  string
	Alias string
	Type  string // class, function, const
	Pos   Position
}

func (u *UseNode) NodeType() string    { return "Use" }
func (u *UseNode) GetPos() Position    { return u.Pos }
func (u *UseNode) SetPos(pos Position) { u.Pos = pos }
func (u *UseNode) String() string {
	if u.Alias != "" {
		return fmt.Sprintf("use %s as %s @ %d:%d", u.Path, u.Alias, u.Pos.Line, u.Pos.Column)
	}
	return fmt.Sprintf("use %s @ %d:%d", u.Path, u.Pos.Line, u.Pos.Column)
}
func (u *UseNode) TokenLiteral() string { return "use" }

// TraitNode represents a PHP trait definition
type TraitNode struct {
	Name    string
	Methods []Node
	Pos     Position
}

func (t *TraitNode) NodeType() string    { return "Trait" }
func (t *TraitNode) GetPos() Position    { return t.Pos }
func (t *TraitNode) SetPos(pos Position) { t.Pos = pos }
func (t *TraitNode) String() string {
	return fmt.Sprintf("trait %s @ %d:%d", t.Name, t.Pos.Line, t.Pos.Column)
}
func (t *TraitNode) TokenLiteral() string { return "trait" }

// MatchNode represents a PHP 8.0+ match expression
type MatchNode struct {
	Condition Node
	Arms      []MatchArmNode
	Pos       Position
}

func (m *MatchNode) NodeType() string    { return "Match" }
func (m *MatchNode) GetPos() Position    { return m.Pos }
func (m *MatchNode) SetPos(pos Position) { m.Pos = pos }
func (m *MatchNode) String() string {
	return fmt.Sprintf("match @ %d:%d", m.Pos.Line, m.Pos.Column)
}
func (m *MatchNode) TokenLiteral() string { return "match" }

// MatchArmNode represents a single arm in a match expression
type MatchArmNode struct {
	Conditions []Node
	Body       Node
	Pos        Position
}

func (m *MatchArmNode) NodeType() string    { return "MatchArm" }
func (m *MatchArmNode) GetPos() Position    { return m.Pos }
func (m *MatchArmNode) SetPos(pos Position) { m.Pos = pos }
func (m *MatchArmNode) String() string {
	return fmt.Sprintf("match arm @ %d:%d", m.Pos.Line, m.Pos.Column)
}
func (m *MatchArmNode) TokenLiteral() string { return "=>" }

// ArrowFunctionNode represents a PHP arrow function (fn)
type ArrowFunctionNode struct {
	Params     []Node
	ReturnType string
	Expr       Node
	Pos        Position
}

func (a *ArrowFunctionNode) NodeType() string    { return "ArrowFunction" }
func (a *ArrowFunctionNode) GetPos() Position    { return a.Pos }
func (a *ArrowFunctionNode) SetPos(pos Position) { a.Pos = pos }
func (a *ArrowFunctionNode) String() string {
	return fmt.Sprintf("fn @ %d:%d", a.Pos.Line, a.Pos.Column)
}
func (a *ArrowFunctionNode) TokenLiteral() string { return "fn" }

// TypeCastNode represents a type cast operation
type TypeCastNode struct {
	Type string
	Expr Node
	Pos  Position
}

func (t *TypeCastNode) NodeType() string    { return "TypeCast" }
func (t *TypeCastNode) GetPos() Position    { return t.Pos }
func (t *TypeCastNode) SetPos(pos Position) { t.Pos = pos }
func (t *TypeCastNode) String() string {
	return fmt.Sprintf("(%s) @ %d:%d", t.Type, t.Pos.Line, t.Pos.Column)
}
func (t *TypeCastNode) TokenLiteral() string { return t.Type }

// YieldNode represents a yield expression
type YieldNode struct {
	Key   Node
	Value Node
	From  bool
	Pos   Position
}

func (y *YieldNode) NodeType() string    { return "Yield" }
func (y *YieldNode) GetPos() Position    { return y.Pos }
func (y *YieldNode) SetPos(pos Position) { y.Pos = pos }
func (y *YieldNode) String() string {
	if y.From {
		return fmt.Sprintf("yield from @ %d:%d", y.Pos.Line, y.Pos.Column)
	}
	return fmt.Sprintf("yield @ %d:%d", y.Pos.Line, y.Pos.Column)
}
func (y *YieldNode) TokenLiteral() string { return "yield" }

// HeredocNode represents a heredoc string
type HeredocNode struct {
	Identifier string
	Parts      []Node
	Pos        Position
}

func (h *HeredocNode) NodeType() string    { return "Heredoc" }
func (h *HeredocNode) GetPos() Position    { return h.Pos }
func (h *HeredocNode) SetPos(pos Position) { h.Pos = pos }
func (h *HeredocNode) String() string {
	return fmt.Sprintf("<<<'%s' @ %d:%d", h.Identifier, h.Pos.Line, h.Pos.Column)
}
func (h *HeredocNode) TokenLiteral() string { return h.Identifier }
