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

// ParameterNode represents a function parameter
type ParameterNode struct {
	Name         string
	TypeHint     string // Type hint for the parameter (e.g., string, int, array)
	DefaultValue Node   // Optional default value
	Visibility   string // public, protected, private (for promoted constructor params)
	IsPromoted   bool   // true if this param is promoted to a property
	IsVariadic   bool   // true if this param is variadic (...$values)
	IsByRef      bool   // true if this param is passed by reference (&$data)
	Pos          Position
}

func (p *ParameterNode) NodeType() string    { return "Parameter" }
func (p *ParameterNode) GetPos() Position    { return p.Pos }
func (p *ParameterNode) SetPos(pos Position) { p.Pos = pos }
func (p *ParameterNode) String() string {
	var parts []string
	if p.Visibility != "" {
		parts = append(parts, p.Visibility)
	}
	if p.TypeHint != "" {
		parts = append(parts, p.TypeHint)
	}
	if p.IsByRef {
		parts = append(parts, "&")
	}
	parts = append(parts, p.Name)
	if p.DefaultValue != nil {
		parts = append(parts, "=", p.DefaultValue.String())
	}
	if p.IsPromoted {
		parts = append(parts, "[promoted]")
	}
	return fmt.Sprintf("%s @ %d:%d", strings.Join(parts, " "), p.Pos.Line, p.Pos.Column)
}
func (p *ParameterNode) TokenLiteral() string {
	return p.Name
}
