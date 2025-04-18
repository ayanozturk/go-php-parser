package tests

import (
	"testing"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
)

func TestParseConstructorPromotionReadonly(t *testing.T) {
	input := `<?php
class Bar {
    public function __construct(
        public readonly string $foo,
        protected readonly ?Baz $baz = null,
        private $plain,
        $notPromoted = 123
    ) {}
}`

	l := lexer.New(input)
	p := parser.New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least one node, got none")
	}

	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("Expected ClassNode, got %T", nodes[0])
	}

	if len(classNode.Methods) == 0 {
		t.Fatalf("Expected at least one method in class")
	}

	ctor, ok := classNode.Methods[0].(*ast.FunctionNode)
	if !ok || ctor.Name != "__construct" {
		t.Fatalf("Expected constructor method, got %T, name=%v", classNode.Methods[0], ctor.Name)
	}

	params := ctor.Params
	if len(params) != 4 {
		t.Fatalf("Expected 4 parameters, got %d", len(params))
	}
}
