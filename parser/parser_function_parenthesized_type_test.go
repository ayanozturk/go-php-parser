package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParseFunctionWithParenthesizedTypeParam(t *testing.T) {
	input := `<?php
class X {
    public function setParent((NodeDefinition&ParentNodeDefinitionInterface)|null $parent): static {}
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
