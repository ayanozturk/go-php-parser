package visitor

import (
	"go-phpcs/ast"
)

// NodeVisitor is the interface for classes that want to visit nodes
type NodeVisitor interface {
	// EnterNode is called before a node's children are traversed
	EnterNode(node ast.Node) bool

	// LeaveNode is called after a node's children are traversed
	LeaveNode(node ast.Node) ast.Node
}

// NodeTraverser is the interface for traversing an AST
type NodeTraverser interface {
	// Traverse traverses an AST using the visitor
	Traverse(node ast.Node, visitor NodeVisitor) ast.Node
}

// BaseTraverser implements basic traversal functionality
type BaseTraverser struct{}

// Traverse traverses an AST using the visitor
func (t *BaseTraverser) Traverse(node ast.Node, visitor NodeVisitor) ast.Node {
	// First enter the node
	if !visitor.EnterNode(node) {
		return visitor.LeaveNode(node)
	}

	// Visit children
	switch n := node.(type) {
	case *ast.Program:
		for i, stmt := range n.Statements {
			n.Statements[i] = t.Traverse(stmt, visitor).(ast.Stmt)
		}
		// Add cases for other node types
	}

	// Leave the node
	return visitor.LeaveNode(node)
}
