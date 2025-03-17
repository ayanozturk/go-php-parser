package ast

// Expr is the interface implemented by all expression nodes
type Expr interface {
	Node
	exprNode()
}

// Expression is the base struct for all expression nodes
type Expression struct {
	BaseNode
}

// exprNode is implemented by all expression nodes to satisfy the Expr interface
func (e *Expression) exprNode() {}
