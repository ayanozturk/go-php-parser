package syntax

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestMaterializeTriviaLiteralsSkipsNoopCopy(t *testing.T) {
	src := []byte("  // c\nx")
	filled := []token.Token{
		{Type: token.T_WHITESPACE, Literal: "  ", Pos: token.Position{Offset: 0}, End: token.Position{Offset: 2}},
		{Type: token.T_COMMENT, Literal: "// c\n", Pos: token.Position{Offset: 2}, End: token.Position{Offset: 7}},
	}
	got := materializeTriviaLiterals(src, filled)
	if &got[0] != &filled[0] {
		t.Fatalf("expected noop reuse of trivia slice when Literals already set")
	}

	empty := []token.Token{
		{Type: token.T_WHITESPACE, Literal: "", Pos: token.Position{Offset: 0}, End: token.Position{Offset: 2}},
	}
	got2 := materializeTriviaLiterals(src, empty)
	if &got2[0] == &empty[0] {
		t.Fatalf("expected copy when Literal needs materialization")
	}
	if got2[0].Literal != "  " {
		t.Fatalf("Literal=%q want %q", got2[0].Literal, "  ")
	}
	if empty[0].Literal != "" {
		t.Fatalf("input slice must not be mutated")
	}
}

func TestOpaqueSpanTokenSharesByWidth(t *testing.T) {
	in := NewInterner(nil)
	a := in.OpaqueSpanToken(12)
	b := in.OpaqueSpanToken(12)
	c := in.OpaqueSpanToken(13)
	if a != b {
		t.Fatal("same width opaque spans must intern")
	}
	if a == c {
		t.Fatal("different widths must not share")
	}
	if !a.IsToken() || a.Width() != 12 {
		t.Fatalf("opaque span IsToken=%v width=%d", a.IsToken(), a.Width())
	}
}

func TestParseForIndexOpaqueBodyIdentity(t *testing.T) {
	src := "<?php\nclass C {\n  public function f($x) {\n    echo \"{$x}\";\n  }\n  private function g() {}\n}\nfunction h() { return 1; }\n"
	res := ParseForIndex([]byte(src))
	if got := Print(res.File.Root); got != src {
		t.Fatalf("ParseForIndex identity\nwant %q\ngot  %q", src, got)
	}
	var bodies int
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if n.Kind() == KindTokenList {
			bodies++
			ch := n.Children()
			if len(ch) != 1 || ch[0].Green == nil || !ch[0].Green.IsToken() {
				t.Fatalf("expected single opaque token body, got %d children", len(ch))
			}
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	if bodies < 3 {
		t.Fatalf("expected >=3 TokenList bodies, got %d", bodies)
	}
	nodes, diags := ParseASTForIndex([]byte(src))
	if len(diags) > 0 {
		t.Fatalf("diags: %v", diags)
	}
	if len(nodes) == 0 {
		t.Fatal("expected lowered nodes")
	}
}

func TestInternerSharesPunctuationAcrossPositions(t *testing.T) {
	src := []byte("()")
	in := NewInterner(src)
	// Two lparens at different absolute offsets must share one green.
	a := in.Token(token.Token{Type: token.T_LPAREN, Literal: "(", Pos: token.Position{Offset: 0}, End: token.Position{Offset: 1}})
	b := in.Token(token.Token{Type: token.T_LPAREN, Literal: "(", Pos: token.Position{Offset: 99}, End: token.Position{Offset: 100}})
	if a != b {
		t.Fatalf("expected shared green for '(' at different offsets")
	}
	if a.token != nil && (a.token.Pos.Offset != 0 || a.token.End.Offset != 0) {
		t.Fatalf("green token must not keep absolute offsets: %+v", a.token.Pos)
	}
}

func TestInternerSharesIdenticalNodes(t *testing.T) {
	in := NewInterner(nil)
	lparen := in.Token(token.Token{Type: token.T_LPAREN, Literal: "("})
	rparen := in.Token(token.Token{Type: token.T_RPAREN, Literal: ")"})
	n1 := in.Node(KindParenExpr, lparen, rparen)
	n2 := in.Node(KindParenExpr, lparen, rparen)
	if n1 != n2 {
		t.Fatalf("expected structural node interning for identical children")
	}
}

func TestInternerDoesNotShareDifferentLiteralsSameWidth(t *testing.T) {
	in := NewInterner(nil)
	a := in.Token(token.Token{Type: token.T_STRING, Literal: "ab"})
	b := in.Token(token.Token{Type: token.T_STRING, Literal: "cd"})
	if a == b {
		t.Fatalf("different literals must not share a green token")
	}
}

func TestInternerSharesModifierListShape(t *testing.T) {
	in := NewInterner(nil)
	pub := in.Token(token.Token{Type: token.T_PUBLIC, Literal: "public"})
	stat := in.Token(token.Token{Type: token.T_STATIC, Literal: "static"})
	m1 := in.Node(KindModifierList, pub, stat)
	m2 := in.Node(KindModifierList, pub, stat)
	if m1 != m2 {
		t.Fatalf("expected shared ModifierList green")
	}
}

func TestGreenFieldsUnexportedImmutableAPI(t *testing.T) {
	in := NewInterner(nil)
	g := in.Token(token.Token{Type: token.T_SEMICOLON, Literal: ";"})
	if g.Kind() != KindToken {
		t.Fatalf("Kind()=%s", g.Kind())
	}
	if g.Width() != 1 {
		t.Fatalf("Width()=%d", g.Width())
	}
	if g.TokenType() != token.T_SEMICOLON {
		t.Fatalf("TokenType()=%s", g.TokenType())
	}
	tok, ok := g.Token()
	if !ok || tok.Literal != ";" {
		t.Fatalf("Token()=%v ok=%v", tok, ok)
	}
}

func TestPrintUsesRedAbsoluteOffsets(t *testing.T) {
	src := []byte("<?php\n$a = 1;\n$b = 1;\n")
	res := Parse(src)
	got := Print(res.File.Root)
	if got != string(src) {
		t.Fatalf("identity\nwant %q\ngot  %q", src, got)
	}
	in := NewInterner(nil)
	oneA := in.Token(token.Token{Type: token.T_LNUMBER, Literal: "1", Pos: token.Position{Offset: 10}, End: token.Position{Offset: 11}})
	oneB := in.Token(token.Token{Type: token.T_LNUMBER, Literal: "1", Pos: token.Position{Offset: 20}, End: token.Position{Offset: 21}})
	if oneA != oneB {
		t.Fatalf("identical '1' literals at different offsets must share green")
	}
	// Subtree print via red absolute offsets (shared green, different Offset).
	f := &File{Source: src, Green: oneA}
	r1 := &RedNode{File: f, Green: oneA, Offset: 10}
	r2 := &RedNode{File: f, Green: oneA, Offset: 20}
	// Place '1' bytes at those offsets in a synthetic buffer.
	buf := make([]byte, 21)
	copy(buf, src)
	buf[10], buf[20] = '1', '1'
	f.Source = buf
	if Print(r1) != "1" || Print(r2) != "1" {
		t.Fatalf("shared green reprint via red offsets failed: %q %q", Print(r1), Print(r2))
	}
}
