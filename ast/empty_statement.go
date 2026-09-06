package ast

import "fmt"

// EmptyStatementNode is a standalone semicolon, including control-structure
// bodies written as `if (cond);`.
type EmptyStatementNode struct {
	Pos    Position
	EndPos Position
}

func (e *EmptyStatementNode) NodeType() string       { return "EmptyStatement" }
func (e *EmptyStatementNode) GetPos() Position       { return e.Pos }
func (e *EmptyStatementNode) SetPos(pos Position)    { e.Pos = pos }
func (e *EmptyStatementNode) GetEndPos() Position    { return e.EndPos }
func (e *EmptyStatementNode) SetEndPos(pos Position) { e.EndPos = pos }
func (e *EmptyStatementNode) String() string {
	return fmt.Sprintf("EmptyStatement @ %d:%d", e.Pos.Line, e.Pos.Column)
}
func (e *EmptyStatementNode) TokenLiteral() string { return ";" }
