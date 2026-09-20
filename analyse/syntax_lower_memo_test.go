package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestMemoLowerFunctionDeclCrossWalk(t *testing.T) {
	src := []byte(`<?php class C { function a(){ return 1; } function b(){ return 2; } }`)
	res := syntax.Parse(src)
	ctx := &AnalysisContext{Content: src, Parsed: res}

	var methodA *syntax.RedNode
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			if methodA == nil {
				methodA = n
			}
		}
		return true
	})
	if methodA == nil {
		t.Fatal("expected a method decl in fixture")
	}

	f1 := memoLowerFunctionDecl(ctx, methodA, res.File)
	methodA2 := &syntax.RedNode{File: res.File, Green: methodA.Green, Offset: methodA.Offset}
	f2 := memoLowerFunctionDecl(ctx, methodA2, res.File)
	if f1 != f2 {
		t.Fatal("expected memo hit across RedNode wrappers with same green+offset")
	}
	if f1 == nil {
		t.Fatal("expected lowered function node")
	}
}

func TestMemoLowerSharedAcrossRuleWalks(t *testing.T) {
	src := []byte(`<?php class C { function a(){ return 1; } function b(){ return 2; } }`)
	res := syntax.Parse(src)
	ctx := &AnalysisContext{Content: src, Parsed: res}

	_ = checkReturnTypeIssuesFromParsed("t.php", res, ctx)
	_ = checkMissingTypeIssuesFromParsed("t.php", res, ctx)

	memo := ensureSyntaxLowerMemo(ctx)
	if memo == nil {
		t.Fatal("expected memo on ctx after rule walks")
	}
	if got, want := len(memo.fnDecl), 2; got != want {
		t.Fatalf("fnDecl memo entries: got %d want %d (one per method, shared across walks)", got, want)
	}
	if got, want := len(memo.classLike), 1; got != want {
		t.Fatalf("classLike memo entries: got %d want %d", got, want)
	}
}

func TestMemoLowerClearedOnSharedReParse(t *testing.T) {
	src := []byte(`<?php class C { function a() {} }`)
	ctx := &AnalysisContext{Content: src}
	res1 := sharedParseResult(ctx, src)
	if res1 == nil {
		t.Fatal("expected parse result")
	}
	var method *syntax.RedNode
	syntax.Walk(res1.File.Root, func(n *syntax.RedNode) bool {
		if n.Kind() == syntax.KindFunctionDecl || n.Kind() == syntax.KindMethodDecl {
			method = n
			return false
		}
		return true
	})
	if method == nil {
		t.Fatal("expected method in fixture")
	}
	_ = memoLowerFunctionDecl(ctx, method, res1.File)
	if ctx.syntaxLower == nil {
		t.Fatal("expected memo populated")
	}

	ctx.Parsed = nil
	src2 := []byte(`<?php class D { function b() {} }`)
	ctx.Content = src2
	res2 := sharedParseResult(ctx, src2)
	if res2 == nil || res2 == res1 {
		t.Fatal("expected new parse result after clearing Parsed")
	}
	if ctx.syntaxLower != nil {
		t.Fatal("expected syntaxLower cleared when ctx.Parsed is replaced")
	}
}
