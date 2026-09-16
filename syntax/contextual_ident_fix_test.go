package syntax

import (
	"strings"
	"testing"
	"time"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestKeywordMemberNamesAfterArrow(t *testing.T) {
	// WordPress-style contextual keywords used as property names after ->.
	// Lexer still emits keyword kinds; parser must accept them as members.
	keywords := []string{
		"extends", "implements", "trait", "insteadof", "type",
		"match", "list", "class", "function", "namespace",
	}
	var b strings.Builder
	b.WriteString("<?php\n")
	for _, kw := range keywords {
		b.WriteString("$o->")
		b.WriteString(kw)
		b.WriteString(";\n")
	}
	src := b.String()
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("identity\nwant %q\ngot  %q", src, got)
	}
	counts := countKinds(res.File.Root, KindMemberAccessExpr, KindTraitDecl)
	if counts[KindMemberAccessExpr] != len(keywords) {
		t.Fatalf("expected %d MemberAccessExpr, got %d", len(keywords), counts[KindMemberAccessExpr])
	}
	if counts[KindTraitDecl] != 0 {
		t.Fatal("trait keyword after -> must not start a TraitDecl")
	}
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, "expected T_LPAREN") ||
			strings.Contains(d.Message, "expected T_LBRACE") ||
			strings.Contains(d.Message, "expected T_RBRACE") {
			t.Fatalf("unexpected recovery diagnostic: %s", d.Message)
		}
	}
}

func TestNamedArgMatchKeyword(t *testing.T) {
	src := "<?php f(match: false);\n"
	done := make(chan struct{})
	var res *ParseResult
	go func() {
		res = Parse([]byte(src))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Parse hung on named arg match: false (match recovery loop)")
	}
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("identity\nwant %q\ngot  %q", src, got)
	}
	counts := countKinds(res.File.Root, KindNamedArg, KindMatchExpr)
	if counts[KindNamedArg] != 1 {
		t.Fatalf("expected KindNamedArg for match: false, tree kinds NamedArg=%d MatchExpr=%d",
			counts[KindNamedArg], counts[KindMatchExpr])
	}
	if counts[KindMatchExpr] != 0 {
		t.Fatal("match: as named arg must not parse as MatchExpr")
	}
}

func TestNamedArgOtherContextualKeywords(t *testing.T) {
	src := "<?php f(class: 1, list: 2, fn: 3);\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("identity\nwant %q\ngot  %q", src, got)
	}
	counts := countKinds(res.File.Root, KindNamedArg)
	if counts[KindNamedArg] != 3 {
		t.Fatalf("expected 3 NamedArg, got %d", counts[KindNamedArg])
	}
}

func TestSwitchCaseNestedBraceBody(t *testing.T) {
	// ManifestSerializer-style: case body opens a block; its } must not close switch.
	src := "<?php\nswitch ($x) {\ncase 1: { echo 1; }\ncase 2: echo 2;\ndefault: echo 0;\n}\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("identity\nwant %q\ngot  %q", src, got)
	}
	counts := countKinds(res.File.Root, KindSwitchStmt, KindCaseClause, KindDefaultClause)
	if counts[KindSwitchStmt] != 1 {
		t.Fatalf("expected 1 SwitchStmt, got %d", counts[KindSwitchStmt])
	}
	if counts[KindCaseClause] != 2 {
		t.Fatalf("expected 2 CaseClause inside switch, got %d", counts[KindCaseClause])
	}
	if counts[KindDefaultClause] != 1 {
		t.Fatalf("expected 1 DefaultClause inside switch, got %d", counts[KindDefaultClause])
	}
}

func TestMatchExprRecoveryProgress(t *testing.T) {
	// Malformed match must terminate; progress guard bumps stuck arms.
	src := "<?php match $x { : };"
	done := make(chan struct{})
	var res *ParseResult
	go func() {
		res = Parse([]byte(src))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Parse hung on malformed match (no progress guard)")
	}
	_ = res
}

func TestInterfaceNamespaceSegmentRoundTrip(t *testing.T) {
	cases := []string{
		"<?php\nuse App\\Module\\Interface\\Foo;\n",
		"<?php\nnamespace App\\Module\\Interface;\n",
		"<?php\nnew \\Vendor\\Pkg\\Class\\Trait\\Interface\\Service();\n",
	}
	for _, src := range cases {
		res := Parse([]byte(src))
		got := Print(res.File.Root)
		if got != src {
			t.Fatalf("identity for %q\nwant %q\ngot  %q", src, src, got)
		}
		for _, d := range res.Diagnostics {
			if strings.Contains(d.Message, "expected name part after") || strings.Contains(d.Message, "Parser.ExpectedToken") {
				t.Fatalf("unexpected diagnostic for %q: %s", src, d.Message)
			}
		}
	}
}

func TestIsContextualIdent(t *testing.T) {
	cases := []struct {
		tt   token.TokenType
		lit  string
		want bool
	}{
		{token.T_STRING, "foo", true},
		{token.T_EXTENDS, "extends", true},
		{token.T_IMPLEMENTS, "implements", true},
		{token.T_TRAIT, "trait", true},
		{token.T_INSTEADOF, "insteadof", true},
		{token.T_MATCH, "match", true},
		{token.T_CLASS, "class", true},
		{token.T_INTERFACE, "Interface", true},
		{token.T_LPAREN, "(", false},
		{token.T_COLON, ":", false},
		{token.T_VARIABLE, "$x", false},
		{token.T_EXTENDS, "", false},
	}
	for _, tc := range cases {
		if got := isContextualIdent(tc.tt, tc.lit); got != tc.want {
			t.Fatalf("isContextualIdent(%v, %q)=%v want %v", tc.tt, tc.lit, got, tc.want)
		}
	}
}
