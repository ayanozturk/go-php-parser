package ast

// Stmt is the interface implemented by all statement nodes
type Stmt interface {
	Node
	stmtNode()
}

// Statement is the base struct for all statement nodes
type Statement struct {
	BaseNode
}

// stmtNode is implemented by all statement nodes to satisfy the Stmt interface
func (s *Statement) stmtNode() {}
