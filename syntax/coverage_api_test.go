package syntax

import (
	"context"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestParseInvocationCount(t *testing.T) {
	ResetParseInvocationCount()
	if ParseInvocationCount() != 0 {
		t.Fatalf("expected 0 after reset, got %d", ParseInvocationCount())
	}
	_ = Parse([]byte("<?php echo 1;"))
	if ParseInvocationCount() == 0 {
		t.Fatal("Parse should increment invocation count")
	}
	before := ParseInvocationCount()
	_ = Parse([]byte("<?php echo 2;"))
	if ParseInvocationCount() != before+1 {
		t.Fatalf("expected %d, got %d", before+1, ParseInvocationCount())
	}
}

func TestIsContextualIdentExported(t *testing.T) {
	if !IsContextualIdent(token.T_STRING, "foo") {
		t.Fatal("T_STRING should be contextual")
	}
	if !IsContextualIdent(token.T_LIST, "list") {
		t.Fatal("reserved word list should be contextual as a member name")
	}
	if IsContextualIdent(token.T_LPAREN, "(") {
		t.Fatal("punctuation must not be contextual")
	}
	if IsContextualIdent(token.T_LIST, "") {
		t.Fatal("empty literal keyword should not be contextual")
	}
}

func TestDecodeStringLiteral(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`'a\'b\\c'`, `a'b\c`},
		{`"a\nb\t\"c"`, "a\nb\t\"c"},
		{`plain`, `plain`},
		{`""`, ``},
		{`''`, ``},
	}
	for _, tc := range cases {
		if got := DecodeStringLiteral(tc.in); got != tc.want {
			t.Fatalf("DecodeStringLiteral(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestPrintGreenAndNilEdges(t *testing.T) {
	src := []byte("<?php echo 1;")
	res := Parse(src)
	if Print(nil) != "" {
		t.Fatal("Print(nil) should be empty")
	}
	if PrintGreen(nil, src) != "" {
		t.Fatal("PrintGreen(nil) should be empty")
	}
	if got := PrintGreen(res.File.Green, src); got != string(src) {
		t.Fatalf("PrintGreen full file mismatch:\n got %q\nwant %q", got, src)
	}
	if PrintGreenAt(nil, src, 0) != "" {
		t.Fatal("PrintGreenAt(nil) should be empty")
	}
}

func TestRedNodeChildForEachPosTokens(t *testing.T) {
	src := []byte("<?php $x = 1;")
	res := Parse(src)
	root := res.File.Root
	if root == nil {
		t.Fatal("nil root")
	}
	var visited int
	root.ForEachChild(func(c *RedNode) bool {
		visited++
		return true
	})
	if visited == 0 {
		t.Fatal("ForEachChild should visit children")
	}
	(*RedNode)(nil).ForEachChild(func(*RedNode) bool { t.Fatal("nil receiver"); return true })
	if root.Child(-1) != nil || root.Child(10_000) != nil {
		t.Fatal("Child out of range should be nil")
	}
	if ch := root.Child(0); ch == nil {
		t.Fatal("expected first child")
	}
	pos := root.Pos()
	end := root.EndPos()
	if pos.Line < 1 || end.Line < 1 {
		t.Fatalf("unexpected positions %#v %#v", pos, end)
	}
	toks := root.Tokens()
	if len(toks) == 0 {
		t.Fatal("expected tokens under root")
	}
	if (*RedNode)(nil).Tokens() != nil {
		t.Fatal("nil Tokens should be nil/empty")
	}
}

func TestSpanAndInternerLen(t *testing.T) {
	if (Span{Start: 2, End: 5}).Len() != 3 {
		t.Fatal("Span.Len")
	}
	if (*Interner)(nil).Len() != 0 {
		t.Fatal("nil Interner.Len")
	}
	in := NewInterner([]byte("abc"))
	_ = in.Token(token.Token{Type: token.T_STRING, Literal: "abc"})
	if in.Len() == 0 {
		t.Fatal("expected interned nodes")
	}
}

func TestParseAndLowerAdapters(t *testing.T) {
	src := []byte("<?php function f($a = 1) { return $a; }\n")
	nodes, res := ParseAndLower(src)
	if res == nil || res.File == nil || len(nodes) == 0 {
		t.Fatalf("ParseAndLower failed: nodes=%d res=%v", len(nodes), res)
	}
	ctxNodes, diags := ParseASTForIndexWithContext(context.Background(), src)
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(ctxNodes) == 0 {
		t.Fatal("expected index AST nodes")
	}
	if LowerAST(nil) != nil {
		t.Fatal("LowerAST(nil) should be nil")
	}
}

func TestAppendUseAliases(t *testing.T) {
	src := []byte(`<?php
namespace App;
use Foo\Bar as Baz;
use function Foo\fn_a;
use const Foo\CONST_A;
use Vendor\{A, function bf, const C};
`)
	res := Parse(src)
	aliases := map[string]string{}
	AppendUseAliases(nil, aliases)
	AppendUseAliases(firstNodeOfKind(res.File.Root, KindUseDecl), nil)

	classA := map[string]string{}
	fnA := map[string]string{}
	constA := map[string]string{}
	Walk(res.File.Root, func(n *RedNode) bool {
		if n.Kind() == KindUseDecl {
			AppendUseAliases(n, classA)
			AppendTypedUseAliases(n, classA, fnA, constA)
		}
		return true
	})
	if classA["baz"] == "" && classA["bar"] == "" {
		// Bar as Baz should land under baz
		if _, ok := classA["baz"]; !ok {
			t.Fatalf("expected class alias, got %#v", classA)
		}
	}
	if classA["baz"] != `Foo\Bar` {
		t.Fatalf("expected Foo\\Bar aliased as baz, got %#v", classA)
	}
	if fnA["fn_a"] == "" && fnA["bf"] == "" {
		t.Fatalf("expected function aliases, got %#v", fnA)
	}
	if constA["const_a"] == "" && constA["c"] == "" {
		t.Fatalf("expected const aliases, got %#v", constA)
	}

	ns, nsAliases := NamespaceAndAliases(res.File)
	if ns != "App" {
		t.Fatalf("namespace=%q", ns)
	}
	if nsAliases["baz"] != `Foo\Bar` {
		t.Fatalf("NamespaceAndAliases class map: %#v", nsAliases)
	}
	emptyNS, emptyAliases := NamespaceAndAliases(nil)
	if emptyNS != "" || len(emptyAliases) != 0 {
		t.Fatalf("nil file namespace: %q %#v", emptyNS, emptyAliases)
	}
}

func TestTernaryElvisCondition(t *testing.T) {
	res := Parse([]byte("<?php $a = $b ?: $c; $d = $e ? $f : $g;\n"))
	var elvis, full *RedNode
	Walk(res.File.Root, func(n *RedNode) bool {
		if n.Kind() != KindTernaryExpr {
			return true
		}
		if green, _ := TernaryElvisCondition(n); green != nil {
			elvis = copyRed(n)
		} else {
			full = copyRed(n)
		}
		return true
	})
	if elvis == nil {
		t.Fatal("expected elvis ternary")
	}
	if green, off := TernaryElvisCondition(elvis); green == nil || off < 0 {
		t.Fatal("elvis should report condition green")
	}
	if green, _ := TernaryElvisCondition(full); green != nil {
		t.Fatal("full ternary should not be elvis")
	}
	if green, _ := TernaryElvisCondition(nil); green != nil {
		t.Fatal("nil ternary")
	}
}

func TestTypeTextAndNameTextNil(t *testing.T) {
	if NameText(nil) != "" || TypeText(nil) != "" {
		t.Fatal("nil NameText/TypeText")
	}
	if UnqualifiedTail(`Ns\Foo`) != "Foo" || UnqualifiedTail("Bar") != "Bar" {
		t.Fatal("UnqualifiedTail")
	}
	if !IsNameKind(KindQualifiedName) || IsNameKind(KindCallExpr) {
		t.Fatal("IsNameKind")
	}
	if !IsTypeKind(KindNamedType) || IsTypeKind(KindCallExpr) {
		t.Fatal("IsTypeKind")
	}
}
