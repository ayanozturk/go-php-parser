package style

import (
	"go-phpcs/ast"
)

type DisallowLongArraySyntaxSniff struct {
	Issues []StyleIssue
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
			s.Issues = append(s.Issues, StyleIssue{
				Filename: filename,
				Line:     arr.GetPos().Line,
				Column:   arr.GetPos().Column,
				Type:     Error,
				Fixable:  false, // not auto-fixable yet
				Message:  "Usage of long array syntax (array(...)) is disallowed; use short syntax ([...]) instead.",
				Code:     "Generic.Arrays.DisallowLongArraySyntax",
				// FixCode: "" // explicitly empty for non-fixable
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

func init() {
	RegisterRule("Generic.Arrays.DisallowLongArraySyntax", func(filename string, _ []byte, nodes []ast.Node) []StyleIssue {
		sniff := &DisallowLongArraySyntaxSniff{}
		sniff.Check(nodes, filename)
		return sniff.Issues
	})
}
