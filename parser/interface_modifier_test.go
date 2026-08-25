package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestParseReorderedInterfaceMethodModifiers(t *testing.T) {
	nodes := parseNoErrors(t, `<?php interface Builder { static public function create(): self; }`)
	interfaceNode := nodes[0].(*ast.InterfaceNode)
	method, ok := interfaceNode.Members[0].(*ast.InterfaceMethodNode)
	if !ok {
		t.Fatalf("expected InterfaceMethodNode, got %T", interfaceNode.Members[0])
	}
	if method.Visibility != "public" || len(method.Modifiers) != 2 || method.Modifiers[0] != "static" || method.Modifiers[1] != "public" {
		t.Fatalf("interface method modifiers were not preserved: %#v", method)
	}
}

func TestParseFinalInterfaceConstant(t *testing.T) {
	nodes := parseNoErrors(t, `<?php interface Priorities { final public const int LOW = 1; }`)
	interfaceNode := nodes[0].(*ast.InterfaceNode)
	constant, ok := interfaceNode.Members[0].(*ast.ConstantNode)
	if !ok {
		t.Fatalf("expected ConstantNode, got %T", interfaceNode.Members[0])
	}
	if constant.Type != "int" || constant.Visibility != "public" || len(constant.Modifiers) != 2 || constant.Modifiers[0] != "final" || constant.Modifiers[1] != "public" {
		t.Fatalf("interface constant modifiers were not preserved: %#v", constant)
	}
}
