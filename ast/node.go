package ast

import (
	"fmt"
)

// Node is the interface that all node types must implement
type Node interface {
	// GetAttributes returns the attributes of this node
	GetAttributes() map[string]interface{}

	// GetType returns the node type as a string
	GetType() string

	// GetStartLine returns the line the node starts on
	GetStartLine() int

	// GetEndLine returns the line the node ends on
	GetEndLine() int

	// String returns a string representation of the node
	String() string
}

// BaseNode provides common functionality for all nodes
type BaseNode struct {
	Attributes map[string]interface{}
	StartLine  int
	EndLine    int
}

// GetAttributes returns the attributes of this node
func (n *BaseNode) GetAttributes() map[string]interface{} {
	return n.Attributes
}

// GetStartLine returns the line the node starts on
func (n *BaseNode) GetStartLine() int {
	return n.StartLine
}

// GetEndLine returns the line the node ends on
func (n *BaseNode) GetEndLine() int {
	return n.EndLine
}

// String returns a string representation of the node
func (n *BaseNode) String() string {
	return fmt.Sprintf("%v", n.Attributes)
}
