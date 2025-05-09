package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseDoublePipeOperator(t *testing.T) {
	php := `<?php
trait Mixin {
    public function nullOrString($value, $message = ''): bool
    {
        $value == null || $message = '';
    }
}`
	l := lexer.New(php)
	p := New(l, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		fmt.Printf("AST (empty): %#v\n", nodes)
		t.Fatal("No nodes parsed from trait")
	} else {
		fmt.Printf("AST: %#v\n", nodes)
	}
	// Find the trait node and check method
	traitFound := false
	methodFound := false
	for _, node := range nodes {
		if trait, ok := node.(*ast.TraitNode); ok {
			traitFound = true
			for _, m := range trait.Body {
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
