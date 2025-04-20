package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseDoublePipeOperator(t *testing.T) {
	php := `<?php
trait Mixin {
    public function nullOrString($value, $message = ''): bool
    {
        null === $value || $message = '';
    }
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes parsed from trait")
	}
	// Find the trait node and check method
	traitFound := false
	methodFound := false
	for _, node := range nodes {
		if trait, ok := node.(*ast.TraitNode); ok {
			traitFound = true
			for _, m := range trait.Methods {
				if fn, ok := m.(*ast.FunctionNode); ok && fn.Name == "nullOrString" {
					methodFound = true
					if len(fn.Params) != 2 {
						t.Errorf("Expected 2 parameters, got %d", len(fn.Params))
					}
					// Check body contains the expected logic
					if len(fn.Body) == 0 {
						t.Errorf("Expected function body, got none")
					}
					// Check for BinaryExpr with operator '||'
					foundOr := false
					for _, stmt := range fn.Body {
						if exprStmt, ok := stmt.(*ast.ExpressionStmt); ok {
							if bin, ok := exprStmt.Expr.(*ast.BinaryExpr); ok {
								if bin.Operator == "||" {
									foundOr = true
									break
								}
							}
						}
					}
					if !foundOr {
						t.Errorf("Expected function body to contain BinaryExpr with operator '||'")
					}
				}
			}
		}
	}
	if !traitFound {
		t.Error("Trait Mixin not found")
	}
	if !methodFound {
		t.Error("nullOrString method not found in trait")
	}
}
