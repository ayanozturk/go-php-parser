package ast

type Node interface {
	NodeType() string
}

type VariableNode struct {
	Name string
}

func (v *VariableNode) NodeType() string { return "Variable" }

type FunctionNode struct {
	Name   string
	Params []Node
	Body   []Node
}

func (f *FunctionNode) NodeType() string { return "Function" }
