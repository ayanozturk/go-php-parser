package parser

import (
	"go-phpcs/lexer"
	"testing"
)

func TestParseMultilineDocCommentInInterface(t *testing.T) {
	php := `<?php
interface NodeVisitorInterface {
    /**
     * Called before child nodes are visited.
     *
     * @return Node The modified node
     */
    public function enterNode(Node $node, Environment $env): Node;
}`
	l := lexer.New(php)
	p := New(l, false)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
}
