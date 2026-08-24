package ast

type ClassConstFetchNode struct {
	Class string
	Const string
	// ConstExpr holds the dynamic name expression for `Class::${expr}`
	// (variable-variable style static property access). Nil for the
	// common `Class::CONST` / `Class::$prop` forms.
	ConstExpr Node
	Pos       Position
	EndPos    Position
}

func (n *ClassConstFetchNode) GetPos() Position {
	return n.Pos
}

func (n *ClassConstFetchNode) NodeType() string {
	return "ClassConstFetchNode"
}

func (n *ClassConstFetchNode) SetPos(pos Position) {
	n.Pos = pos
}

func (n *ClassConstFetchNode) GetEndPos() Position {
	return n.EndPos
}

func (n *ClassConstFetchNode) SetEndPos(pos Position) {
	n.EndPos = pos
}

func (n *ClassConstFetchNode) String() string {
	return n.Class + "::" + n.Const
}

func (n *ClassConstFetchNode) TokenLiteral() string {
	return n.Class + "::" + n.Const
}
