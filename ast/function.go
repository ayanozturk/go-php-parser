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
	Uses       []ClosureUse
	Body       []Node
	PHPDoc     *PHPDocNode // Associated PHPDoc comment
	Pos        Position
	EndPos     Position
	// HeaderEndPos marks the end of the declaration's signature/header
	// (i.e. right after the return type, or after the closing ')' of the
	// parameter list when there is no return type), before the body's '{'.
	// Diagnostics about the declaration itself (missing types, invalid
	// modifiers, etc.) should end here rather than at EndPos so they don't
	// underline the entire method/function body.
	HeaderEndPos Position
}

// ClosureUse preserves one explicit anonymous-function capture. By-reference
// captures define the outer variable at closure creation; by-value captures
// read the outer variable.
type ClosureUse struct {
	Name   string
	ByRef  bool
	Pos    Position
	EndPos Position
}

func (f *FunctionNode) NodeType() string       { return "Function" }
func (f *FunctionNode) GetPos() Position       { return f.Pos }
func (f *FunctionNode) SetPos(pos Position)    { f.Pos = pos }
func (f *FunctionNode) GetEndPos() Position    { return f.EndPos }
func (f *FunctionNode) SetEndPos(pos Position) { f.EndPos = pos }

// GetHeaderEndPos returns the end of the function/method's signature
// (before its body), falling back to EndPos if HeaderEndPos was never set
// (e.g. nodes constructed outside the parser).
func (f *FunctionNode) GetHeaderEndPos() Position {
	if f.HeaderEndPos == (Position{}) {
		return f.EndPos
	}
	return f.HeaderEndPos
}
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
	Name   Node   // Function name (identifier or variable)
	Args   []Node // Arguments (may include UnpackedArgumentNode)
	Pos    Position
	EndPos Position
}

func (f *FunctionCallNode) NodeType() string       { return "FunctionCall" }
func (f *FunctionCallNode) GetPos() Position       { return f.Pos }
func (f *FunctionCallNode) SetPos(pos Position)    { f.Pos = pos }
func (f *FunctionCallNode) GetEndPos() Position    { return f.EndPos }
func (f *FunctionCallNode) SetEndPos(pos Position) { f.EndPos = pos }
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
	Expr   Node
	Pos    Position
	EndPos Position
}

func (u *UnpackedArgumentNode) NodeType() string       { return "UnpackedArgument" }
func (u *UnpackedArgumentNode) GetPos() Position       { return u.Pos }
func (u *UnpackedArgumentNode) SetPos(pos Position)    { u.Pos = pos }
func (u *UnpackedArgumentNode) GetEndPos() Position    { return u.EndPos }
func (u *UnpackedArgumentNode) SetEndPos(pos Position) { u.EndPos = pos }
func (u *UnpackedArgumentNode) String() string {
	if u.Expr == nil {
		return "...<nil>"
	}
	return fmt.Sprintf("...%s", u.Expr.String())
}
func (u *UnpackedArgumentNode) TokenLiteral() string { return "..." }

type NamedArgumentNode struct {
	Name   string
	Value  Node
	Pos    Position
	EndPos Position
}

func (n *NamedArgumentNode) NodeType() string       { return "NamedArgument" }
func (n *NamedArgumentNode) GetPos() Position       { return n.Pos }
func (n *NamedArgumentNode) SetPos(pos Position)    { n.Pos = pos }
func (n *NamedArgumentNode) GetEndPos() Position    { return n.EndPos }
func (n *NamedArgumentNode) SetEndPos(pos Position) { n.EndPos = pos }
func (n *NamedArgumentNode) String() string {
	if n.Value == nil {
		return fmt.Sprintf("%s: <nil>", n.Name)
	}
	return fmt.Sprintf("%s: %s", n.Name, n.Value.String())
}
func (n *NamedArgumentNode) TokenLiteral() string { return n.Name }
