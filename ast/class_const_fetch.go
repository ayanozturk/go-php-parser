package ast

type ClassConstFetchNode struct {
	Class string
	Const string
	Pos   Position
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

func (n *ClassConstFetchNode) String() string {
	return n.Class + "::" + n.Const
}

func (n *ClassConstFetchNode) TokenLiteral() string {
	return n.Class + "::" + n.Const
}



