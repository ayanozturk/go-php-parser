package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestParseClosureUseListPreservesNamesReferencesAndSpans(t *testing.T) {
	p := New(lexer.New(`<?php $closure = function () use ($value, &$reference) {};`), false)
	nodes := p.Parse()
	if len(p.Errors()) != 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	var closure *ast.FunctionNode
	walkParserNodes(nodes, func(node ast.Node) {
		if function, ok := node.(*ast.FunctionNode); ok && function.Name == "" {
			closure = function
		}
	})
	if closure == nil {
		t.Fatal("anonymous closure was not preserved")
	}
	if len(closure.Uses) != 2 {
		t.Fatalf("closure uses = %#v, want two captures", closure.Uses)
	}
	if got := closure.Uses[0]; got.Name != "value" || got.ByRef || got.EndPos.Offset <= got.Pos.Offset {
		t.Fatalf("by-value capture = %#v", got)
	}
	if got := closure.Uses[1]; got.Name != "reference" || !got.ByRef || got.EndPos.Offset <= got.Pos.Offset {
		t.Fatalf("by-reference capture = %#v", got)
	}
}

func walkParserNodes(nodes []ast.Node, visit func(ast.Node)) {
	for _, node := range nodes {
		visit(node)
		switch n := node.(type) {
		case *ast.ExpressionStmt:
			walkParserExpression(n.Expr, visit)
		case *ast.FunctionNode:
			walkParserNodes(n.Body, visit)
		}
	}
}

func walkParserExpression(node ast.Node, visit func(ast.Node)) {
	if node == nil {
		return
	}
	visit(node)
	switch n := node.(type) {
	case *ast.AssignmentNode:
		walkParserExpression(n.Left, visit)
		walkParserExpression(n.Right, visit)
	case *ast.FunctionNode:
		walkParserNodes(n.Body, visit)
	}
}
