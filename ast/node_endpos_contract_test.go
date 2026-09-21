package ast

import "testing"

// TestNodeEndPosContract exercises GetEndPos/SetEndPos on representative nodes
// that already have String/TokenLiteral coverage elsewhere. Diagnostics and
// span-sensitive analyse paths rely on these setters.
func TestNodeEndPosContract(t *testing.T) {
	start := Position{Line: 1, Column: 1, Offset: 0}
	end := Position{Line: 1, Column: 8, Offset: 7}
	nodes := []Node{
		&BlockNode{Pos: start, EndPos: end},
		&Identifier{Name: "x", Pos: start, EndPos: end},
		&VariableNode{Name: "v", Pos: start, EndPos: end},
		&StringLiteral{Value: "s", Pos: start, EndPos: end},
		&IntegerLiteral{Value: 1, Pos: start, EndPos: end},
		&FloatLiteral{Value: 1.5, Pos: start, EndPos: end},
		&BooleanNode{Value: true, Pos: start, EndPos: end},
		&NullNode{Pos: start, EndPos: end},
		&InterpolatedStringLiteral{Pos: start, EndPos: end},
		&HeredocNode{Identifier: "EOT", Pos: start, EndPos: end},
		&YieldNode{Pos: start, EndPos: end},
		&TypeCastNode{Type: "int", Pos: start, EndPos: end},
		&UnionTypeNode{Pos: start, EndPos: end},
		&IntersectionTypeNode{Pos: start, EndPos: end},
		&ParamNode{Name: "p", Pos: start, EndPos: end},
		&FunctionNode{Name: "f", Pos: start, EndPos: end},
		&FunctionCallNode{Pos: start, EndPos: end},
		&FunctionCall{Name: "f", Pos: start, EndPos: end},
		&FunctionDecl{Name: "f", Pos: start, EndPos: end},
		&Variable{Name: "v", Pos: start, EndPos: end},
		&IdentifierNode{Value: "id", Pos: start, EndPos: end},
		&ClassNode{Name: "C", Pos: start, EndPos: end},
		&PropertyNode{Name: "p", Pos: start, EndPos: end},
		&EnumNode{Name: "E", Pos: start, EndPos: end},
		&EnumCaseNode{Name: "A", Pos: start, EndPos: end},
		&InterfaceNode{Name: "I", Pos: start, EndPos: end},
		&InterfaceMethodNode{Name: "m", Pos: start, EndPos: end},
		&CommentNode{Value: "//", Pos: start, EndPos: end},
		&ClassConstFetchNode{Class: "C", Const: "X", Pos: start, EndPos: end},
		&ArrayNode{Pos: start, EndPos: end},
		&KeyValueNode{Pos: start, EndPos: end},
		&ArrayAccessNode{Var: &VariableNode{Name: "a"}, Index: &IntegerLiteral{Value: 0}, Pos: start, EndPos: end},
		&ArrayItemNode{Pos: start, EndPos: end},
		&CallableParamNode{Pos: start, EndPos: end},
		&CallableTypeNode{Pos: start, EndPos: end},
		&UnaryExpr{Pos: start, EndPos: end},
		&ConstantNode{Name: "C", Pos: start, EndPos: end},
		&DeclareNode{Pos: start, EndPos: end},
		&StringNode{Value: "s", Pos: start, EndPos: end},
		&IntegerNode{Value: 1, Pos: start, EndPos: end},
		&FloatNode{Value: 1.5, Pos: start, EndPos: end},
		&NamedArgumentNode{Name: "a", Pos: start, EndPos: end},
		&UnpackedArgumentNode{Pos: start, EndPos: end},
		&MethodCallNode{Method: "m", Pos: start, EndPos: end},
		&NewNode{ClassName: "C", Pos: start, EndPos: end},
		&TraitNode{Pos: start, EndPos: end},
		&ThrowNode{Expr: &IntegerLiteral{Value: 1}, Pos: start, EndPos: end},
		&DoWhileNode{Condition: &BooleanNode{Value: true}, Pos: start, EndPos: end},
		&FirstClassCallableNode{Name: &IdentifierNode{Value: "f"}, Pos: start, EndPos: end},
	}

	for _, n := range nodes {
		n := n
		t.Run(n.NodeType(), func(t *testing.T) {
			assertLoopControlSpan(t, n, start, end)
		})
	}
}
