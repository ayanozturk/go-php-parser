package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
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

func TestClassMemberDeclarationsHaveCompleteSpans(t *testing.T) {
	src := `<?php
class Model {
    public string $name;
    public const KIND = 'model';
    public function label(int $id): string { return (string) $id; }
}
`
	p := New(lexer.NewFile(src), false)
	nodes := p.Parse()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	class, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected class node, got %T", nodes[0])
	}
	declarations := []ast.Node{class.Properties[0], class.Constants[0], class.Methods[0]}
	for _, declaration := range declarations {
		assertCompleteNodeSpan(t, src, declaration)
	}
}

func TestSkippedFunctionBodyRetainsCompleteDeclarationSpan(t *testing.T) {
	src := `<?php
class Model {
    public function label(): string {
        return "}";
    }
    public function next(): void {}
}
`
	p := New(lexer.NewFile(src), false)
	p.SkipFunctionBodies = true
	nodes := p.Parse()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	class, ok := nodes[0].(*ast.ClassNode)
	if !ok || len(class.Methods) != 2 {
		t.Fatalf("expected class with two methods, got %#v", nodes)
	}
	first := class.Methods[0]
	assertCompleteNodeSpan(t, src, first)
	if got := src[first.GetPos().Offset:first.GetEndPos().Offset]; got != "function label(): string {\n        return \"}\";\n    }" {
		t.Fatalf("skipped method span mismatch: %q", got)
	}
	assertCompleteNodeSpan(t, src, class.Methods[1])
}

func assertCompleteNodeSpan(t *testing.T, source string, node ast.Node) {
	t.Helper()
	start, end := node.GetPos(), node.GetEndPos()
	if start.Offset < 0 || end.Offset <= start.Offset || end.Offset > len(source) {
		t.Fatalf("invalid %s span: start=%+v end=%+v source length=%d", node.NodeType(), start, end, len(source))
	}
}
