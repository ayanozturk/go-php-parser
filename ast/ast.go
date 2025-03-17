package ast

import (
	"fmt"
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

// LiteralNode represents a literal value like number or string
type LiteralNode struct {
	Value string
	Pos   Position
}

func (l *LiteralNode) NodeType() string    { return "Literal" }
func (l *LiteralNode) GetPos() Position    { return l.Pos }
func (l *LiteralNode) SetPos(pos Position) { l.Pos = pos }
func (l *LiteralNode) String() string {
	return fmt.Sprintf("Literal(%s) @ %d:%d", l.Value, l.Pos.Line, l.Pos.Column)
}
func (l *LiteralNode) TokenLiteral() string {
	return l.Value
}

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

// FunctionNode represents a PHP function definition
type FunctionNode struct {
	Name   string
	Params []Node
	Body   []Node
	Pos    Position
}

func (f *FunctionNode) NodeType() string    { return "Function" }
func (f *FunctionNode) GetPos() Position    { return f.Pos }
func (f *FunctionNode) SetPos(pos Position) { f.Pos = pos }
func (f *FunctionNode) String() string {
	return fmt.Sprintf("Function(%s) @ %d:%d", f.Name, f.Pos.Line, f.Pos.Column)
}
func (f *FunctionNode) TokenLiteral() string {
	return "function"
}

// ParameterNode represents a function parameter
type ParameterNode struct {
	Name         string
	Type         string
	ByRef        bool
	Variadic     bool
	DefaultValue Node
	Pos          Position
}

func (p *ParameterNode) NodeType() string    { return "Parameter" }
func (p *ParameterNode) GetPos() Position    { return p.Pos }
func (p *ParameterNode) SetPos(pos Position) { p.Pos = pos }
func (p *ParameterNode) String() string {
	var parts []string
	if p.Type != "" {
		parts = append(parts, p.Type)
	}
	if p.ByRef {
		parts = append(parts, "&")
	}
	if p.Variadic {
		parts = append(parts, "...")
	}
	parts = append(parts, fmt.Sprintf("$%s", p.Name))
	if p.DefaultValue != nil {
		parts = append(parts, "=", p.DefaultValue.String())
	}
	return fmt.Sprintf("Parameter(%s) @ %d:%d", strings.Join(parts, " "), p.Pos.Line, p.Pos.Column)
}
func (p *ParameterNode) TokenLiteral() string {
	return p.Name
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

// IfNode represents an if-else statement
type IfNode struct {
	Condition Node
	ThenBlock []Node
	ElseBlock []Node
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

// PrintAST recursively prints the AST tree with indentation
func PrintAST(nodes []Node, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	for _, node := range nodes {
		fmt.Println(prefix + node.String())
		switch n := node.(type) {
		case *FunctionNode:
			PrintAST(n.Params, indent+1)
			PrintAST(n.Body, indent+1)
		case *AssignmentNode:
			PrintAST([]Node{n.Left, n.Right}, indent+1)
		case *BinaryExpr:
			PrintAST([]Node{n.Left, n.Right}, indent+1)
		case *ReturnNode:
			PrintAST([]Node{n.Expr}, indent+1)
		case *ExpressionStmt:
			PrintAST([]Node{n.Expr}, indent+1)
		case *IfNode:
			PrintAST([]Node{n.Condition}, indent+1)
			fmt.Println(prefix + "  Then:")
			PrintAST(n.ThenBlock, indent+2)
			if len(n.ElseBlock) > 0 {
				fmt.Println(prefix + "  Else:")
				PrintAST(n.ElseBlock, indent+2)
			}
		case *WhileNode:
			PrintAST([]Node{n.Condition}, indent+1)
			PrintAST(n.Body, indent+1)
		}
	}
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
