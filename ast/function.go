package ast

import (
	"fmt"
	"strings"
)

// FunctionNode represents a PHP function definition
type FunctionNode struct {
	Name       string
	Visibility string   // public, private, protected (legacy, kept for compatibility)
	Modifiers  []string // All modifiers, e.g. public, static, final, abstract
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
	if len(f.Modifiers) > 0 {
		parts = append(parts, strings.Join(f.Modifiers, " "))
	}
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
// (e.g., sprintf($format ?? ”, ...$values))
type FunctionCallNode struct {
	Name Node   // Function name (identifier or variable)
	Args []Node // Arguments (may include UnpackedArgumentNode)
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
	nameStr := "<nil>"
	if f.Name != nil {
		nameStr = f.Name.String()
	}
	return fmt.Sprintf("FunctionCall(%s, [%s]) @ %d:%d", nameStr, strings.Join(argStrs, ", "), f.Pos.Line, f.Pos.Column)
}
func (f *FunctionCallNode) TokenLiteral() string {
	if f.Name != nil {
		return f.Name.TokenLiteral()
	}
	return ""
}

// UnpackedArgumentNode represents ...$values in function call arguments
type UnpackedArgumentNode struct {
	Expr Node
	Pos  Position
}

func (u *UnpackedArgumentNode) NodeType() string    { return "UnpackedArgument" }
func (u *UnpackedArgumentNode) GetPos() Position    { return u.Pos }
func (u *UnpackedArgumentNode) SetPos(pos Position) { u.Pos = pos }
func (u *UnpackedArgumentNode) String() string {
	if u.Expr == nil {
		return "...<nil>"
	}
	return fmt.Sprintf("...%s", u.Expr.String())
}
func (u *UnpackedArgumentNode) TokenLiteral() string { return "..." }
