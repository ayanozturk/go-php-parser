package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestParseEmptyStatements(t *testing.T) {
	source := `<?php
;
if ($x);
while ($drain) ;
for ($i = 0; $i < 10; $i++) ;
`
	p := New(lexer.NewFile(source), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parser errors: %v", errs)
	}
	if len(nodes) != 4 {
		t.Fatalf("top-level node count = %d, want 4", len(nodes))
	}
	if _, ok := nodes[0].(*ast.EmptyStatementNode); !ok {
		t.Fatalf("first node = %T, want EmptyStatementNode", nodes[0])
	}
	ifNode, ok := nodes[1].(*ast.IfNode)
	if !ok || len(ifNode.Body) != 1 {
		t.Fatalf("if node = %#v, want IfNode with one body statement", nodes[1])
	}
	if _, ok := ifNode.Body[0].(*ast.EmptyStatementNode); !ok {
		t.Fatalf("if body = %T, want EmptyStatementNode", ifNode.Body[0])
	}
	whileNode, ok := nodes[2].(*ast.WhileNode)
	if !ok || len(whileNode.Body) != 1 {
		t.Fatalf("while node = %#v, want WhileNode with one body statement", nodes[2])
	}
	if _, ok := whileNode.Body[0].(*ast.EmptyStatementNode); !ok {
		t.Fatalf("while body = %T, want EmptyStatementNode", whileNode.Body[0])
	}
	forNode, ok := nodes[3].(*ast.ForNode)
	if !ok || len(forNode.Body) != 1 {
		t.Fatalf("for node = %#v, want ForNode with one body statement", nodes[3])
	}
	if _, ok := forNode.Body[0].(*ast.EmptyStatementNode); !ok {
		t.Fatalf("for body = %T, want EmptyStatementNode", forNode.Body[0])
	}
}
