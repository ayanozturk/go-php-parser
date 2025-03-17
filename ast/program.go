package ast

// Program represents the root node of the AST
type Program struct {
	BaseNode
	Statements []Stmt
}

// GetType returns the node type
func (p *Program) GetType() string {
	return "Program"
}
