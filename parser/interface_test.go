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

func TestParseInterfaceWithStaticReturnType(t *testing.T) {
	php := `<?php
interface StaticReturnTypeInterface {
    public function foo(): static;
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes returned from parser")
	}
	iface, ok := nodes[0].(*ast.InterfaceNode)
	if !ok {
		t.Fatalf("Expected InterfaceNode, got %T", nodes[0])
	}
	if len(iface.Methods) == 0 {
		t.Fatal("Expected at least one method in interface")
	}
	method, ok := iface.Methods[0].(*ast.InterfaceMethodNode)
	if !ok {
		t.Fatalf("Expected InterfaceMethodNode, got %T", iface.Methods[0])
	}
	retType, ok := method.ReturnType.(*ast.IdentifierNode)
	if !ok || retType.Value != "static" {
		t.Errorf("Expected return type 'static', got %v", method.ReturnType)
	}
}

func TestParseInterfaceWithUnionTypeAndFQCN(t *testing.T) {
	php := `<?php
interface Foo {
    public function bar(): string|\Stringable;
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	iface, ok := nodes[0].(*ast.InterfaceNode)
	if !ok {
		t.Fatalf("Expected InterfaceNode, got %T", nodes[0])
	}
	if len(iface.Methods) == 0 {
		t.Fatal("Expected at least one method in interface")
	}
	method, ok := iface.Methods[0].(*ast.InterfaceMethodNode)
	if !ok {
		t.Fatalf("Expected InterfaceMethodNode, got %T", iface.Methods[0])
	}
	union, ok := method.ReturnType.(*ast.UnionTypeNode)
	if !ok {
		t.Fatalf("Expected UnionTypeNode, got %T", method.ReturnType)
	}
	if len(union.Types) != 2 || union.Types[0] != "string" || union.Types[1] != "\\Stringable" {
		t.Errorf("Expected union types [string \\Stringable], got %v", union.Types)
	}
}
