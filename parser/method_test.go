package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestInterfaceExtendsFQCN(t *testing.T) {
	input := `<?php
interface MyInterface extends \Some\OtherInterface {}`
	l := lexer.New(input)
	// DEBUG: Dump tokens
	tokens := []string{}
	for i := 0; i < 20; i++ {
		tok := l.NextToken()
		tokens = append(tokens, string(tok.Type)+" (\""+tok.Literal+"\")")
		if tok.Type == "T_EOF" {
			break
		}
	}
	t.Logf("TOKENS: %v", tokens)
	l = lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser returned errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes parsed from input")
	}
	iface, ok := nodes[0].(*ast.InterfaceNode)
	if !ok {
		t.Fatalf("Expected InterfaceNode, got %T", nodes[0])
	}
	if iface.Name != "MyInterface" {
		t.Errorf("Expected interface name 'MyInterface', got '%s'", iface.Name)
	}
	if len(iface.Extends) != 1 {
		t.Fatalf("Expected 1 extended interface, got %d", len(iface.Extends))
	}
	if iface.Extends[0] != "\\Some\\OtherInterface" {
		t.Errorf("Expected extends to be '\\Some\\OtherInterface', got '%s'", iface.Extends[0])
	}
}
