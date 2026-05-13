package parser

import (
	"testing"

	"go-phpcs/ast"
	"go-phpcs/lexer"
)

func TestParseUseDeclaration(t *testing.T) {
	code := "<?php\nuse App\\Domain\\Notification;\n"
	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	useNode, ok := nodes[0].(*ast.UseNode)
	if !ok {
		t.Fatalf("expected UseNode, got %T", nodes[0])
	}
	if useNode.Path != "App\\Domain\\Notification" {
		t.Fatalf("expected path App\\Domain\\Notification, got %q", useNode.Path)
	}
	if useNode.Alias != "Notification" {
		t.Fatalf("expected alias Notification, got %q", useNode.Alias)
	}
	if useNode.Type != "class" {
		t.Fatalf("expected class use type, got %q", useNode.Type)
	}
}

func TestParseUseDeclarationWithAlias(t *testing.T) {
	code := "<?php\nuse App\\Domain\\Notification as MessageNotification;\n"
	l := lexer.New(code)
	p := New(l, true)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	useNode := nodes[0].(*ast.UseNode)
	if useNode.Alias != "MessageNotification" {
		t.Fatalf("expected alias MessageNotification, got %q", useNode.Alias)
	}
}
