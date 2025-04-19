package tests

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"testing"
)

func TestParseFunctionWithUnionTypeParam(t *testing.T) {
	input := `<?php
function foo(\DOMException|\Dom\Exception $e, array $a, Stub $stub, bool $isNested) {}`

	l := lexer.New(input)
	p := parser.New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}
}
