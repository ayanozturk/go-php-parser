package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParseInterfaceWithNullableReturnType(t *testing.T) {
	php := `<?php
interface NodeVisitorInterface {
    public function enterNode(Node $node, Environment $env): Node;
    public function leaveNode(Node $node, Environment $env): ?Node;
}`
	l := lexer.New(php)
	p := New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}
