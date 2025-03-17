package ast

// Error represents a parsing error
type Error struct {
	BaseNode
	Message string
}

// GetType returns the node type
func (e *Error) GetType() string {
	return "Error"
}

// String returns a string representation of the error
func (e *Error) String() string {
	return e.Message
}
