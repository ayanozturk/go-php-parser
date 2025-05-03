package arrays

import (
	"go-phpcs/ast"
	"go-phpcs/style"
)

// DisallowLongArraySyntaxSniff checks for usage of long array syntax (array(...)).
type DisallowLongArraySyntaxSniff struct {
	Issues []style.StyleIssue
}

func (s *DisallowLongArraySyntaxSniff) Check(nodes []ast.Node, filename string) {
	for _, node := range nodes {
		s.checkNode(node, filename)
	}
}

func (s *DisallowLongArraySyntaxSniff) checkNode(node ast.Node, filename string) {
	if node == nil {
		return
	}
	if arr, ok := node.(*ast.ArrayNode); ok {
		if arr.TokenLiteral() == "array" {
			s.Issues = append(s.Issues, style.StyleIssue{
				Filename: filename,
				Line:     arr.GetPos().Line,
				Column:   arr.GetPos().Column,
				Type:     style.Error,
				Fixable:  true,
				Message:  "Usage of long array syntax (array(...)) is disallowed; use short syntax ([...]) instead.",
				Code:     "Generic.Arrays.DisallowLongArraySyntax",
			})
		}
		s.checkArrayNodeChildren(arr, filename)
		return
	}
	if item, ok := node.(*ast.ArrayItemNode); ok {
		s.checkArrayItemNodeChildren(item, filename)
		return
	}
	if kv, ok := node.(*ast.KeyValueNode); ok {
		s.checkKeyValueNodeChildren(kv, filename)
		return
	}
	if acc, ok := node.(*ast.ArrayAccessNode); ok {
		s.checkArrayAccessNodeChildren(acc, filename)
		return
	}
}

func (s *DisallowLongArraySyntaxSniff) checkArrayNodeChildren(n *ast.ArrayNode, filename string) {
	for _, el := range n.Elements {
		s.checkNode(el, filename)
	}
}

func (s *DisallowLongArraySyntaxSniff) checkArrayItemNodeChildren(n *ast.ArrayItemNode, filename string) {
	if n.Key != nil {
		s.checkNode(n.Key, filename)
	}
	if n.Value != nil {
		s.checkNode(n.Value, filename)
	}
}

func (s *DisallowLongArraySyntaxSniff) checkKeyValueNodeChildren(n *ast.KeyValueNode, filename string) {
	if n.Key != nil {
		s.checkNode(n.Key, filename)
	}
	if n.Value != nil {
		s.checkNode(n.Value, filename)
	}
}

func (s *DisallowLongArraySyntaxSniff) checkArrayAccessNodeChildren(n *ast.ArrayAccessNode, filename string) {
	if n.Var != nil {
		s.checkNode(n.Var, filename)
	}
	if n.Index != nil {
		s.checkNode(n.Index, filename)
	}
}
