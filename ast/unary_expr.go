package ast

type UnaryExpr struct {
	Operator string
	Operand  Node
	Pos      Position
}

func (u *UnaryExpr) GetPos() Position {
	return u.Pos
}

func (u *UnaryExpr) NodeType() string {
	return "UnaryExpr"
}

func (u *UnaryExpr) SetPos(pos Position) {
	u.Pos = pos
}

func (u *UnaryExpr) String() string {
	return u.Operator + u.Operand.String()
}

func (u *UnaryExpr) TokenLiteral() string {
	return u.Operator
}
