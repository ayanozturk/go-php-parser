package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestInterfaceExtends(t *testing.T) {
	code := `<?php
interface A {}
interface B extends A {}
interface C extends A, B {}
`
	lex := lexer.New(code)
	p := New(lex, false)
	nodes := p.Parse()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	a, ok := nodes[0].(*ast.InterfaceNode)
	if !ok || a.Name != "A" {
		t.Errorf("first node should be interface A, got %T %v", nodes[0], nodes[0])
	}
	if len(a.Extends) != 0 {
		t.Errorf("interface A should not extend anything, got %v", a.Extends)
	}

	b, ok := nodes[1].(*ast.InterfaceNode)
	if !ok || b.Name != "B" {
		t.Errorf("second node should be interface B, got %T %v", nodes[1], nodes[1])
	}
	if len(b.Extends) != 1 || b.Extends[0] != "A" {
		t.Errorf("interface B should extend [A], got %v", b.Extends)
	}

	c, ok := nodes[2].(*ast.InterfaceNode)
	if !ok || c.Name != "C" {
		t.Errorf("third node should be interface C, got %T %v", nodes[2], nodes[2])
	}
	if len(c.Extends) != 2 || c.Extends[0] != "A" || c.Extends[1] != "B" {
		t.Errorf("interface C should extend [A B], got %v", c.Extends)
	}
}
