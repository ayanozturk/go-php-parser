package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseForWithBlock(t *testing.T) {
	php := `<?php
for ($i = 0; $i < 3; $i++) {
    echo "x";
}`
	l := lexer.New(php)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	blk, ok := nodes[0].(*ast.BlockNode)
	if !ok {
		t.Fatalf("Expected BlockNode for for-loop body, got %T", nodes[0])
	}
	if len(blk.Statements) != 1 {
		t.Fatalf("Expected 1 statement in for body, got %d", len(blk.Statements))
	}
	if _, ok := blk.Statements[0].(*ast.ExpressionStmt); !ok {
		t.Fatalf("Expected ExpressionStmt inside for body, got %T", blk.Statements[0])
	}
}

func TestParseForWithoutBlock(t *testing.T) {
	php := `<?php
for ($i = 0; $i < 2; $i++) echo 1;`
	l := lexer.New(php)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}
	blk, ok := nodes[0].(*ast.BlockNode)
	if !ok {
		t.Fatalf("Expected BlockNode for for-loop body, got %T", nodes[0])
	}
	if len(blk.Statements) != 1 {
		t.Fatalf("Expected 1 statement in for body, got %d", len(blk.Statements))
	}
	stmt, ok := blk.Statements[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("Expected ExpressionStmt inside for body, got %T", blk.Statements[0])
	}
	if stmt.Expr == nil || stmt.Expr.TokenLiteral() != "1" {
		t.Fatalf("Expected integer literal '1' expression, got %T with token %q", stmt.Expr, func() string {
			if stmt.Expr != nil {
				return stmt.Expr.TokenLiteral()
			}
			return ""
		}())
	}
}
