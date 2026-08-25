package parser

import (
	"reflect"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func parseForFixture(t *testing.T, source string) *ast.ForNode {
	t.Helper()
	p := New(lexer.NewFile(source), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parser errors: %v", errs)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one top-level node, got %d", len(nodes))
	}
	loop, ok := nodes[0].(*ast.ForNode)
	if !ok {
		t.Fatalf("expected ForNode, got %T", nodes[0])
	}
	return loop
}

func TestParseForWithBlockPreservesControlExpressions(t *testing.T) {
	source := `<?php
for ($index = 0; $index < 3; $index++) {
    echo "tick";
}`
	loop := parseForFixture(t, source)

	if len(loop.Init) != 1 || len(loop.Conditions) != 1 || len(loop.Updates) != 1 {
		t.Fatalf("unexpected control clause sizes: init=%d conditions=%d updates=%d", len(loop.Init), len(loop.Conditions), len(loop.Updates))
	}
	if _, ok := loop.Init[0].(*ast.AssignmentNode); !ok {
		t.Fatalf("initializer expression was not preserved as an assignment: %T", loop.Init[0])
	}
	condition, ok := loop.Conditions[0].(*ast.BinaryExpr)
	if !ok || condition.Operator != "<" {
		t.Fatalf("condition expression was not preserved: %#v", loop.Conditions[0])
	}
	update, ok := loop.Updates[0].(*ast.UnaryExpr)
	if !ok || update.Operator != "++" {
		t.Fatalf("update expression was not preserved: %#v", loop.Updates[0])
	}
	if len(loop.Body) != 1 {
		t.Fatalf("expected one statement in brace body, got %d", len(loop.Body))
	}
	if _, ok := loop.Body[0].(*ast.ExpressionStmt); !ok {
		t.Fatalf("expected expression statement in brace body, got %T", loop.Body[0])
	}
}

func TestParseForPreservesCommaSeparatedClauses(t *testing.T) {
	loop := parseForFixture(t, `<?php for ($left = 0, $right = 1; $left < 4, $right < 5; $left++, $right += 2) echo $left;`)

	if got, want := len(loop.Init), 2; got != want {
		t.Fatalf("initializer count = %d, want %d", got, want)
	}
	if got, want := len(loop.Conditions), 2; got != want {
		t.Fatalf("condition count = %d, want %d", got, want)
	}
	if got, want := len(loop.Updates), 2; got != want {
		t.Fatalf("update count = %d, want %d", got, want)
	}

	if got := loop.Init[0].(*ast.AssignmentNode).Left.(*ast.VariableNode).Name; got != "left" {
		t.Fatalf("first initializer target = %q, want left", got)
	}
	if got := loop.Init[1].(*ast.AssignmentNode).Left.(*ast.VariableNode).Name; got != "right" {
		t.Fatalf("second initializer target = %q, want right", got)
	}
	if got := loop.Updates[0].(*ast.UnaryExpr).Operator; got != "++" {
		t.Fatalf("first update operator = %q, want ++", got)
	}
	if got := loop.Updates[1].(*ast.AssignmentNode).Operator; got != "+=" {
		t.Fatalf("second update operator = %q, want +=", got)
	}
}

func TestParseForAllowsEmptyControlClauses(t *testing.T) {
	loop := parseForFixture(t, `<?php for (;;) { break; }`)

	if loop.Init == nil || loop.Conditions == nil || loop.Updates == nil {
		t.Fatalf("empty clauses should be represented by non-nil empty slices: init=%#v conditions=%#v updates=%#v", loop.Init, loop.Conditions, loop.Updates)
	}
	if !reflect.DeepEqual(loop.Init, []ast.Node{}) || !reflect.DeepEqual(loop.Conditions, []ast.Node{}) || !reflect.DeepEqual(loop.Updates, []ast.Node{}) {
		t.Fatalf("empty clause contents should be empty: init=%#v conditions=%#v updates=%#v", loop.Init, loop.Conditions, loop.Updates)
	}
	if len(loop.Body) != 1 {
		t.Fatalf("expected one statement in empty-clause loop body, got %d", len(loop.Body))
	}
}

func TestParseForWithoutBlockKeepsSingleStatementBody(t *testing.T) {
	loop := parseForFixture(t, `<?php for ($index = 0; $index < 2; $index++) echo 1;`)

	if len(loop.Body) != 1 {
		t.Fatalf("expected one statement in single-statement body, got %d", len(loop.Body))
	}
	stmt, ok := loop.Body[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("expected expression statement in single-statement body, got %T", loop.Body[0])
	}
	if stmt.Expr == nil || stmt.Expr.TokenLiteral() != "1" {
		t.Fatalf("expected integer literal '1' expression, got %T with token %q", stmt.Expr, nodeToken(stmt.Expr))
	}
}

func TestParseForHasCompleteBraceSpan(t *testing.T) {
	source := "<?php\nfor ($step = 0; $step < 2; $step++) {\n    echo $step;\n}"
	loop := parseForFixture(t, source)

	start, end := loop.GetPos(), loop.GetEndPos()
	if got := source[start.Offset:end.Offset]; got != "for ($step = 0; $step < 2; $step++) {\n    echo $step;\n}" {
		t.Fatalf("for span = %q, want complete loop source", got)
	}
	if start.Line != 2 || start.Column != 1 || end.Offset != len(source) {
		t.Fatalf("unexpected for span positions: start=%+v end=%+v", start, end)
	}
}

func nodeToken(node ast.Node) string {
	if node == nil {
		return ""
	}
	return node.TokenLiteral()
}
