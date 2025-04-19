package ast

import (
	"fmt"
	"strings"
)

// FunctionNode represents a PHP function definition
type FunctionNode struct {
	Name       string
	Visibility string // public, private, protected
	ReturnType string
	Params     []Node
	Body       []Node
	Pos        Position
}

func (f *FunctionNode) NodeType() string    { return "Function" }
func (f *FunctionNode) GetPos() Position    { return f.Pos }
func (f *FunctionNode) SetPos(pos Position) { f.Pos = pos }
func (f *FunctionNode) String() string {
	var parts []string
	if f.Visibility != "" {
		parts = append(parts, f.Visibility)
	}
	parts = append(parts, fmt.Sprintf("Function(%s)", f.Name))
	if f.ReturnType != "" {
		parts = append(parts, fmt.Sprintf(": %s", f.ReturnType))
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), f.Pos.Line, f.Pos.Column)
}
func (f *FunctionNode) TokenLiteral() string {
	return "function"
}

// FunctionCallNode represents a function call expression
// (e.g., sprintf($format ?? '', ...$values))
type FunctionCallNode struct {
	Name string    // Function name (identifier)
	Args []Node    // Arguments (may include UnpackedArgumentNode)
	Pos  Position
}

func (f *FunctionCallNode) NodeType() string    { return "FunctionCall" }
func (f *FunctionCallNode) GetPos() Position    { return f.Pos }
func (f *FunctionCallNode) SetPos(pos Position) { f.Pos = pos }
func (f *FunctionCallNode) String() string {
	var argStrs []string
	for _, arg := range f.Args {
		argStrs = append(argStrs, arg.String())
	}
	return fmt.Sprintf("FunctionCall(%s, [%s]) @ %d:%d", f.Name, strings.Join(argStrs, ", "), f.Pos.Line, f.Pos.Column)
}
func (f *FunctionCallNode) TokenLiteral() string { return f.Name }

// UnpackedArgumentNode represents ...$values in function call arguments
type UnpackedArgumentNode struct {
	Expr Node
	Pos  Position
}
func (u *UnpackedArgumentNode) NodeType() string    { return "UnpackedArgument" }
func (u *UnpackedArgumentNode) GetPos() Position    { return u.Pos }
func (u *UnpackedArgumentNode) SetPos(pos Position) { u.Pos = pos }
func (u *UnpackedArgumentNode) String() string      { return fmt.Sprintf("...%s", u.Expr.String()) }
func (u *UnpackedArgumentNode) TokenLiteral() string { return "..." }

