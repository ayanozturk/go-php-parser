package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestParseLoopControlPreservesImplicitAndNumericLevelsAndSpans(t *testing.T) {
	source := `<?php
function advance(): void {
    break;
    break 2;
    continue;
    continue 3;
}`
	p := New(lexer.NewFile(source), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parser errors: %v", errs)
	}
	if len(nodes) != 1 {
		t.Fatalf("top-level node count = %d, want 1", len(nodes))
	}
	function, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("top-level node = %T, want *ast.FunctionNode", nodes[0])
	}
	if len(function.Body) != 4 {
		t.Fatalf("function statement count = %d, want 4", len(function.Body))
	}

	firstBreak, ok := function.Body[0].(*ast.BreakNode)
	if !ok || firstBreak.Level != nil {
		t.Fatalf("first statement = %#v, want implicit BreakNode", function.Body[0])
	}
	assertParsedStatementSpan(t, source, firstBreak, "break;", 3, 5)

	secondBreak, ok := function.Body[1].(*ast.BreakNode)
	if !ok {
		t.Fatalf("second statement = %T, want *ast.BreakNode", function.Body[1])
	}
	assertIntegerLoopLevel(t, secondBreak.Level, 2)
	assertParsedStatementSpan(t, source, secondBreak, "break 2;", 4, 5)

	firstContinue, ok := function.Body[2].(*ast.ContinueNode)
	if !ok || firstContinue.Level != nil {
		t.Fatalf("third statement = %#v, want implicit ContinueNode", function.Body[2])
	}
	assertParsedStatementSpan(t, source, firstContinue, "continue;", 5, 5)

	secondContinue, ok := function.Body[3].(*ast.ContinueNode)
	if !ok {
		t.Fatalf("fourth statement = %T, want *ast.ContinueNode", function.Body[3])
	}
	assertIntegerLoopLevel(t, secondContinue.Level, 3)
	assertParsedStatementSpan(t, source, secondContinue, "continue 3;", 6, 5)
}

func assertIntegerLoopLevel(t *testing.T, node ast.Node, want int64) {
	t.Helper()
	level, ok := node.(*ast.IntegerNode)
	if !ok {
		t.Fatalf("loop level = %T, want *ast.IntegerNode", node)
	}
	if level.Value != want {
		t.Fatalf("loop level = %d, want %d", level.Value, want)
	}
}

func assertParsedStatementSpan(t *testing.T, source string, node ast.Node, want string, line, column int) {
	t.Helper()
	start, end := node.GetPos(), node.GetEndPos()
	if got := source[start.Offset:end.Offset]; got != want {
		t.Fatalf("statement span = %q, want %q", got, want)
	}
	if start.Line != line || start.Column != column {
		t.Fatalf("statement start = %+v, want line %d column %d", start, line, column)
	}
}
