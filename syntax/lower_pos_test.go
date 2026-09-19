package syntax

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func TestNodePosSkipsLeadingTrivia(t *testing.T) {
	src := []byte("<?php\nfunction value(): int\n{\n    return 1;\n    $x = 2; // @phpstan-ignore-line\n}\n")
	res := Parse(src)
	var fnTok *RedNode
	var varTok *RedNode
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if n.Green != nil && n.Green.IsToken() {
			tok, _ := n.Green.Token()
			if tok.Type == token.T_FUNCTION {
				fnTok = n
			}
			if tok.Type == token.T_VARIABLE && tok.Literal == "$x" {
				varTok = n
			}
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	if fnTok == nil || varTok == nil {
		t.Fatalf("tokens not found fn=%v var=%v", fnTok != nil, varTok != nil)
	}
	fp, _ := nodePos(res.File, fnTok)
	t.Logf("function tok span=%v contentStart=%d pos=%d:%d lines=%v", fnTok.Span(), contentStartOffset(fnTok), fp.Line, fp.Column, res.File.Lines)
	if fp.Line != 2 || fp.Column != 1 {
		t.Fatalf("function pos=%d:%d want 2:1", fp.Line, fp.Column)
	}
	vp, _ := nodePos(res.File, varTok)
	t.Logf("var tok span=%v contentStart=%d pos=%d:%d", varTok.Span(), contentStartOffset(varTok), vp.Line, vp.Column)
	if vp.Line != 5 {
		t.Fatalf("var pos=%d:%d want line 5", vp.Line, vp.Column)
	}

	nodes := LowerAST(res)
	fn := nodes[0].(*ast.FunctionNode)
	t.Logf("lowered func pos=%d:%d", fn.Pos.Line, fn.Pos.Column)
	if fn.Pos.Line != 2 {
		t.Fatalf("lowered function pos line=%d want 2", fn.Pos.Line)
	}
}

// TestRuneColumnAtHandlesMultibyteAndBackwardJumps guards the runeColumnAt
// cache added to positionAt: lowering a nested array computes a container's
// own (start, end) before descending into its children, so offsets routinely
// jump backward relative to the last positionAt call on the same line. The
// cache must still return correct rune columns (not just fast ones) under
// that access pattern, including when multibyte runes are present so a byte
// offset and a rune column genuinely differ.
func TestRuneColumnAtHandlesMultibyteAndBackwardJumps(t *testing.T) {
	// One line: ['é', 'é', 'bb', 'é', 'cc'] - mixes multibyte content with
	// plain ASCII so forward/backward jumps land on both kinds of segment.
	src := []byte("<?php\n$x = ['é', 'éé', 'bb', 'é', 'cc'];\n")
	res := Parse(src)

	// Simulate the nested-array traversal pattern directly: query an offset
	// near the end of the line first (as a container's own end position
	// would), then walk backward through earlier offsets (as its children's
	// positions would), and confirm every column matches a from-scratch
	// computation with no cache involved.
	line := 1 // 0-based: the `$x = [...]` line
	lineStart := res.File.Lines[line]
	// Offsets below land exactly on character boundaries in
	// "$x = ['é', 'éé', 'bb', 'é', 'cc'];" (verified against the source byte
	// layout) - real callers only ever query lexer token-boundary offsets,
	// which by construction never split a multibyte rune, so this mirrors
	// what the cache actually sees rather than testing an input it can never
	// receive in practice.
	offsets := []int{37, 32, 26, 5, 12, 2, 30}
	for _, off := range offsets {
		absOffset := lineStart + off
		got := runeColumnAt(res.File, line, lineStart, absOffset)
		want := len([]rune(string(src[lineStart:absOffset])))
		if got != want {
			t.Fatalf("runeColumnAt(offset %d) = %d, want %d (independently computed)", off, got, want)
		}
	}
}

// TestRuneColumnAtFallsBackOnNonBoundaryOffsets covers a case real lexer
// token offsets can't produce but that the cache must still handle exactly
// rather than merely fast: an offset landing mid-rune. This happens for
// genuinely non-UTF-8 PHP source (legacy Latin-1/Windows-1252 string
// literals are common) where a byte can look like a UTF-8 continuation byte
// without the lexer treating it as part of any multi-byte rune. Splitting a
// RuneCount into forward/backward deltas around such an offset is not the
// same computation as one direct scan, so the cache must detect this and
// fall back to a full rescan instead of returning an off-by-one column.
func TestRuneColumnAtFallsBackOnNonBoundaryOffsets(t *testing.T) {
	src := []byte("<?php\n$x = ['é', 'éé', 'bb', 'é', 'cc'];\n")
	res := Parse(src)
	line := 1
	lineStart := res.File.Lines[line]

	// Prime the cache at a clean boundary (end of line), then query an offset
	// that splits the 2-byte 'é' at source offset 7-8 (offset 28 in the full
	// source = lineStart+22, landing on the continuation byte).
	_ = runeColumnAt(res.File, line, lineStart, lineStart+37)
	off := 28 // mid-'é' relative to lineStart, per the byte layout above
	got := runeColumnAt(res.File, line, lineStart, lineStart+off)
	want := len([]rune(string(src[lineStart : lineStart+off])))
	if got != want {
		t.Fatalf("runeColumnAt at non-boundary offset = %d, want %d (direct scan)", got, want)
	}
}

// TestLowerASTFastOnLongSingleLineArray is a regression guard for a real
// O(N^2) bug: positionAt used to rune-count from the start of the current
// line on every call, so a huge single-line array literal (common in
// generated files, e.g. vendor SDK API definitions) made every element's
// position cost proportional to its byte offset within the line, and the
// whole file cost O(N^2). A vendor file of this shape took over 100s to
// lower before the runeColumnAt cache; this synthetic equivalent must stay
// well under a second.
func TestLowerASTFastOnLongSingleLineArray(t *testing.T) {
	var b strings.Builder
	b.WriteString("<?php\nreturn [")
	for i := 0; i < 20000; i++ {
		fmt.Fprintf(&b, "'key%d' => 'value%d', ", i, i)
	}
	b.WriteString("];\n")
	src := []byte(b.String())

	res := Parse(src)
	start := time.Now()
	nodes := LowerAST(res)
	elapsed := time.Since(start)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}
	if elapsed > 2*time.Second {
		t.Fatalf("lowering a 20000-element single-line array took %s, want well under 2s (O(N^2) regression?)", elapsed)
	}
}

// TestReservedWordClassNameParsesAsExpression is a regression test for a
// real bug: isNameStart (which decides whether the current token can start
// a name reference in expression position) only recognized a narrow,
// hardcoded set of token types, missing most PHP reserved words. PHP
// allows a reserved word as a class/constant name reference outside
// declaration position - e.g. doctrine/annotations ships a real class
// literally named Enum (Doctrine\Common\Annotations\Annotation\Enum),
// referenced elsewhere as `Enum::class`. That reference failed to parse
// specifically inside a short array literal ("expected T_RBRACKET, got
// T_ENUM"), even though the identical expression parsed fine as a bare
// statement - isNameStart is only consulted once parsePrimaryExpr's
// explicit per-token cases are exhausted, so the array-element parsing
// path (which reaches the same default branch) was affected the same way
// any other unhandled reserved word would be.
func TestReservedWordClassNameParsesAsExpression(t *testing.T) {
	src := []byte(`<?php
$x = ['enum' => Enum::class, 'target' => Target::class];
`)
	res := Parse(src)
	if len(res.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", res.Diagnostics)
	}
	if got := Print(res.File.Root); got != string(src) {
		t.Fatalf("round-trip identity failed\nwant %q\ngot  %q", src, got)
	}
}

// TestDynamicStaticPropertyAccessParses is a regression test for a real
// bug: parseStaticMemberAccess's switch had no case for a dynamic property
// name after `::` (`self::$$payload`, accessing the static property whose
// *name* is the runtime value of $payload) - only plain `self::$payload`
// was handled. This is the same lexer shape as an ordinary variable
// variable ($$var: T_ILLEGAL("$") then the inner variable), just after
// `::` instead of standalone.
func TestDynamicStaticPropertyAccessParses(t *testing.T) {
	src := []byte(`<?php
class Foo {
    public static $bar = 1;
    public function m($payload) {
        if (isset(self::$$payload)) {
            return self::$$payload;
        }
    }
}
`)
	res := Parse(src)
	if len(res.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", res.Diagnostics)
	}
	if got := Print(res.File.Root); got != string(src) {
		t.Fatalf("round-trip identity failed\nwant %q\ngot  %q", src, got)
	}
}

// TestYieldFromParsesInNestedExpressionContext is a regression test for a
// real bug: T_YIELD_FROM is defined in the token package and referenced in
// the parser's dispatch tables, but no keyword table entry ever produces
// it - the lexer always tokenizes "yield from" as plain T_YIELD followed by
// a separate T_STRING("from") token. parseYieldExpr only consumed the
// first token, so "from" was parsed as the yield's own (bare-name) value
// and whatever followed it was left dangling. Standalone `yield from x();`
// happened to still "work" (both fragments were independently valid
// statements, silently producing the wrong tree - two statements instead
// of one delegated yield), but nesting it inside an expression - e.g. an
// if-condition's assignment, a real pattern in generator-based async code
// (amphp/ReactPHP-style `if (null === $x = yield from gen())`) - surfaced
// a real parse error: "expected T_RPAREN, got T_STRING".
func TestYieldFromParsesInNestedExpressionContext(t *testing.T) {
	src := []byte(`<?php
function gen() {
    if (null === $response = yield from other()) {
        return 1;
    }
}
`)
	res := Parse(src)
	if len(res.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", res.Diagnostics)
	}
	if got := Print(res.File.Root); got != string(src) {
		t.Fatalf("round-trip identity failed\nwant %q\ngot  %q", src, got)
	}
}

// TestYieldFromLowersWithCorrectShape locks in the AST shape a correctly
// parsed "yield from" must produce: From=true, no Key, and Value set to the
// delegated expression - as opposed to the pre-fix behavior of silently
// splitting into two separate statements.
func TestYieldFromLowersWithCorrectShape(t *testing.T) {
	src := []byte(`<?php
function gen() {
    yield from other();
    yield $k => $v;
}
`)
	nodes, diags := ParseAST(src)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
	fn, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("expected *ast.FunctionNode, got %T", nodes[0])
	}
	if len(fn.Body) != 2 {
		t.Fatalf("expected 2 statements in the function body (one yield from, one yield), got %d: %#v", len(fn.Body), fn.Body)
	}

	yieldFrom, ok := fn.Body[0].(*ast.ExpressionStmt).Expr.(*ast.YieldNode)
	if !ok {
		t.Fatalf("expected *ast.YieldNode, got %T", fn.Body[0].(*ast.ExpressionStmt).Expr)
	}
	if !yieldFrom.From {
		t.Fatal("expected From=true for yield from")
	}
	if yieldFrom.Key != nil {
		t.Fatalf("yield from must not have a key, got %#v", yieldFrom.Key)
	}
	if _, ok := yieldFrom.Value.(*ast.FunctionCallNode); !ok {
		t.Fatalf("expected yield from's Value to be the delegated call, got %T", yieldFrom.Value)
	}

	plainYield, ok := fn.Body[1].(*ast.ExpressionStmt).Expr.(*ast.YieldNode)
	if !ok {
		t.Fatalf("expected *ast.YieldNode, got %T", fn.Body[1].(*ast.ExpressionStmt).Expr)
	}
	if plainYield.From {
		t.Fatal("expected From=false for a plain yield $k => $v")
	}
	if plainYield.Key == nil {
		t.Fatal("expected a key for yield $k => $v")
	}
}
