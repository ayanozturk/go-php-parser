package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParseFunctionWithUnionTypeParam(t *testing.T) {
	input := `<?php
function foo(\DOMException|\Dom\Exception $e, array $a, Stub $stub, bool $isNested) {}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}
}

func TestParseFunctionWithAttributeAndUnionReturnType(t *testing.T) {
	input := `<?php
#[SomeAttr]
function foo(string $a): int|string
{
    return 1;
}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}
}

func TestParseFunctionWithComplexUnionAndNullableTypes(t *testing.T) {
	input := `<?php
function bar(array|string|null $x = null, ?string $y = "abc"): array|string|false {}`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}
}
