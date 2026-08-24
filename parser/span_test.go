package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/lexer"
)

// TestParseStatementEndPosSpansStatement verifies that parseStatement's
// wrapper stamps each top-level statement's EndPos to the position right
// after its last consumed token, giving a complete [start, end) source
// span rather than only a start point (see M1's "complete source spans"
// requirement in docs/full-static-analyser-target.md).
func TestParseStatementEndPosSpansStatement(t *testing.T) {
	src := "<?php\nfunction foo($a) {\n    return $a * 2;\n}\n"
	l := lexer.NewFile(src)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}
	fn := nodes[0]
	start := fn.GetPos()
	end := fn.GetEndPos()
	if start.Line != 2 || start.Column != 1 {
		t.Fatalf("unexpected start pos: %+v", start)
	}
	if end.Line != 4 || end.Column != 2 {
		t.Fatalf("unexpected end pos: %+v", end)
	}
	if end.Offset <= start.Offset {
		t.Fatalf("expected end offset > start offset, got start=%d end=%d", start.Offset, end.Offset)
	}
	snippet := src[start.Offset:end.Offset]
	want := "function foo($a) {\n    return $a * 2;\n}"
	if snippet != want {
		t.Fatalf("span snippet mismatch:\n got: %q\nwant: %q", snippet, want)
	}
}

// TestParseExpressionEndPosSpansExpression verifies expression nodes parsed
// via parseExpressionWithPrecedence (the common expression entry point)
// also get a correct EndPos, including nested sub-expressions reached
// recursively through that same wrapped entry point.
func TestParseExpressionEndPosSpansExpression(t *testing.T) {
	src := "<?php\n$x = 1 + 2;\n"
	l := lexer.NewFile(src)
	p := New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}
	stmt := nodes[0]
	end := stmt.GetEndPos()
	if end.Line != 2 {
		t.Fatalf("unexpected end line: %+v", end)
	}
	// The statement's span must reach through to the closing ';', not stop
	// short at some inner sub-expression.
	if src[end.Offset-1] != ';' {
		t.Fatalf("expected span to end just after ';', got byte %q at offset %d", src[end.Offset-1], end.Offset-1)
	}
}
