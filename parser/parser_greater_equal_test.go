package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"testing"
)

func TestParseGreaterEqualOperator(t *testing.T) {
	src := `<?php
class User {
    public $age = 20;
    public function isAdult() {
        return $this->age >= 18;
    }
}`
	lex := lexer.New(src)
	p := New(lex, true)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("Parser errors: %v", p.Errors())
	}
	if len(nodes) == 0 {
		t.Fatal("No nodes parsed from class")
	}
	// Find the class node and check method
	classFound := false
	methodFound := false
	for _, node := range nodes {
		if classNode, ok := node.(*ast.ClassNode); ok {
			classFound = true
			for _, m := range classNode.Methods {
				if fn, ok := m.(*ast.FunctionNode); ok && fn.Name == "isAdult" {
					methodFound = true
					if len(fn.Body) == 0 {
						t.Errorf("Expected function body, got none")
					}
					// Check for ReturnNode with BinaryExpr >=
					foundGE := false
					for _, stmt := range fn.Body {
						if ret, ok := stmt.(*ast.ReturnNode); ok {
							if bin, ok := ret.Expr.(*ast.BinaryExpr); ok {
								if bin.Operator == ">=" {
									foundGE = true
									break
								}
							}
						}
					}
					if !foundGE {
						t.Errorf("Expected BinaryExpr with operator >= in return statement")
					}
				}
			}
		}
	}
	if !classFound {
		t.Error("Class User not found")
	}
	if !methodFound {
		t.Error("Method isAdult not found")
	}
}
