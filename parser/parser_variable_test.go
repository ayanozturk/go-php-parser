package parser

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"testing"
)

func TestExpressionStatementKeepsAssignmentVarDoc(t *testing.T) {
	p := New(lexer.New(`<?php
function run(?User $from): void {
    /** @var User $user */
    $user = $from;
}
`), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parser errors: %v", errs)
	}
	fn, ok := nodes[0].(*ast.FunctionNode)
	if !ok || len(fn.Body) != 1 {
		t.Fatalf("expected function with one statement, got %#v", nodes)
	}
	stmt, ok := fn.Body[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("statement = %T, want ExpressionStmt", fn.Body[0])
	}
	if stmt.PHPDoc == nil || stmt.PHPDoc.VarType != "User" || stmt.PHPDoc.VarName != "user" {
		t.Fatalf("assignment PHPDoc = %#v, want @var User $user", stmt.PHPDoc)
	}
}

func TestExpressionStatementKeepsAssignmentVarDocAfterIf(t *testing.T) {
	p := New(lexer.New(`<?php
function run(array $items, ?User $from): void {
    if (empty($items)) {
        return;
    }
    /** @var User $user */
    $user = $from;
}
`), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parser errors: %v", errs)
	}
	fn, ok := nodes[0].(*ast.FunctionNode)
	if !ok || len(fn.Body) != 2 {
		t.Fatalf("expected function with two statements, got %#v", nodes)
	}
	if _, ok := fn.Body[0].(*ast.IfNode); !ok {
		t.Fatalf("first statement = %T, want IfNode", fn.Body[0])
	}
	stmt, ok := fn.Body[1].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("second statement = %T, want ExpressionStmt", fn.Body[1])
	}
	if stmt.PHPDoc == nil || stmt.PHPDoc.VarType != "User" || stmt.PHPDoc.VarName != "user" {
		t.Fatalf("assignment PHPDoc = %#v, want @var User $user", stmt.PHPDoc)
	}
}

func TestAssignmentNodesPreserveOperator(t *testing.T) {
	for _, operator := range []string{"=", "+=", "??="} {
		t.Run(operator, func(t *testing.T) {
			p := New(lexer.New(fmt.Sprintf("<?php $value %s 1;", operator)), false)
			nodes := p.Parse()
			if errs := p.Errors(); len(errs) != 0 {
				t.Fatalf("parser errors: %v", errs)
			}
			if len(nodes) != 1 {
				t.Fatalf("node count = %d, want 1", len(nodes))
			}
			stmt, ok := nodes[0].(*ast.ExpressionStmt)
			if !ok {
				t.Fatalf("node = %T, want ExpressionStmt", nodes[0])
			}
			assignment, ok := stmt.Expr.(*ast.AssignmentNode)
			if !ok {
				t.Fatalf("expression = %T, want AssignmentNode", stmt.Expr)
			}
			if assignment.Operator != operator {
				t.Fatalf("operator = %q, want %q", assignment.Operator, operator)
			}
		})
	}
}

func TestArrayVariables(t *testing.T) {
	input := `<?php
	$var3 = [1, 2, 3];
	`

	l := lexer.New(input)
	p := New(l, true)
	nodes := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parser returned errors: %v", p.Errors())
	}

	if len(nodes) == 0 {
		t.Error("Expected at least one node, but got none")
	}

	for _, node := range nodes {
		// Accept ExpressionStmt wrapping AssignmentNode as valid
		var assignNode *ast.AssignmentNode
		if exprStmt, ok := node.(*ast.ExpressionStmt); ok {
			if a, ok := exprStmt.Expr.(*ast.AssignmentNode); ok {
				assignNode = a
			} else {
				t.Errorf("Expected AssignmentNode inside ExpressionStmt, found %s", exprStmt.Expr.NodeType())
				continue
			}
		}
		if assignNode == nil {
			// Fallback: maybe it's a bare AssignmentNode (legacy)
			if a, ok := node.(*ast.AssignmentNode); ok {
				assignNode = a
			} else {
				t.Errorf("Expected AssignmentNode (possibly wrapped in ExpressionStmt), found %s", node.NodeType())
				continue
			}
		}

		if _, ok := assignNode.Left.(*ast.VariableNode); !ok {
			t.Errorf("Expected Left to be VariableNode, found %s", assignNode.Left.NodeType())
		}

		arrayNode, ok := assignNode.Right.(*ast.ArrayNode)
		if !ok {
			t.Errorf("Expected Right to be ArrayNode, found %s", assignNode.Right.NodeType())
			continue
		}

		if len(arrayNode.Elements) != 3 {
			t.Errorf("Expected ArrayNode to have 3 elements, but got %d", len(arrayNode.Elements))
		}
	}
}
