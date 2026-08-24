package ast

import "fmt"

type SwitchNode struct {
	Expr   Node
	Cases  []*SwitchCaseNode
	Pos    Position
	EndPos Position
}

func (s *SwitchNode) NodeType() string       { return "Switch" }
func (s *SwitchNode) GetPos() Position       { return s.Pos }
func (s *SwitchNode) SetPos(pos Position)    { s.Pos = pos }
func (s *SwitchNode) GetEndPos() Position    { return s.EndPos }
func (s *SwitchNode) SetEndPos(pos Position) { s.EndPos = pos }
func (s *SwitchNode) String() string {
	return fmt.Sprintf("Switch @ %d:%d", s.Pos.Line, s.Pos.Column)
}
func (s *SwitchNode) TokenLiteral() string { return "switch" }

type SwitchCaseNode struct {
	Expr      Node
	IsDefault bool
	Body      []Node
	Pos       Position
	EndPos    Position
}

func (s *SwitchCaseNode) NodeType() string       { return "SwitchCase" }
func (s *SwitchCaseNode) GetPos() Position       { return s.Pos }
func (s *SwitchCaseNode) SetPos(pos Position)    { s.Pos = pos }
func (s *SwitchCaseNode) GetEndPos() Position    { return s.EndPos }
func (s *SwitchCaseNode) SetEndPos(pos Position) { s.EndPos = pos }
func (s *SwitchCaseNode) String() string {
	if s.IsDefault {
		return fmt.Sprintf("DefaultCase @ %d:%d", s.Pos.Line, s.Pos.Column)
	}
	return fmt.Sprintf("Case @ %d:%d", s.Pos.Line, s.Pos.Column)
}
func (s *SwitchCaseNode) TokenLiteral() string {
	if s.IsDefault {
		return "default"
	}
	return "case"
}
