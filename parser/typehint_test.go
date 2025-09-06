package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseNullableUnionTypeHint(t *testing.T) {
	input := `<?php
function foo(null|\Product\Custom\Entity $customProductEntity) {}`
	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser returned errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes parsed from input")
	}
	funcNode, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("Expected first node to be FunctionNode, got %T", nodes[0])
	}
	if len(funcNode.Params) != 1 {
		t.Fatalf("Expected one parameter, got %d", len(funcNode.Params))
	}
	param, ok := funcNode.Params[0].(*ast.ParamNode)
	if !ok {
		t.Fatalf("Expected parameter to be ParamNode, got %T", funcNode.Params[0])
	}
	expected := "null|\\Product\\Custom\\Entity"
	if param.TypeHint != expected {
		t.Errorf("Expected type hint '%s', got '%s'", expected, param.TypeHint)
	}
}
