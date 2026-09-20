package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// sharedParseFixture mixes a top-level declaration, PSR-1 side effects, and an
// assignment-in-condition so fused CST rules and Content-gated contextual rules
// exercise sharedParseResult when ctx.Content is set.
const sharedParseFixture = `<?php
class C {
	public function run() {}
}
if ($x = foo()) {
	bar();
}
echo 1;
`

// warmProjectStubIndex loads embedded PHP stub definitions once so
// BuildProjectIndex during RunAnalysisRulesWithContext does not inflate
// ParseInvocationCount in tests.
func warmProjectStubIndex() {
	_ = BuildProjectIndex(map[string][]ast.Node{})
}

func TestSharedParseResultOnce(t *testing.T) {
	warmProjectStubIndex()
	syntax.ResetParseInvocationCount()

	src := []byte(sharedParseFixture)
	res := syntax.Parse(src)
	nodes := syntax.LowerAST(res)
	baseline := syntax.ParseInvocationCount()
	if baseline == 0 {
		t.Fatal("expected baseline parse count > 0 after syntax.Parse")
	}

	project := BuildProjectIndex(map[string][]ast.Node{"t.php": nodes})
	// nil AnalysisLevel enables all registered rules, including Content-gated
	// SideEffects / AssignmentInCondition which must reuse ctx.Parsed.
	ctx := &AnalysisContext{
		Content:  src,
		Parsed:   res,
		Resolver: project,
	}
	_ = RunAnalysisRulesWithContext("t.php", nodes, ctx)

	if got := syntax.ParseInvocationCount(); got != baseline {
		t.Fatalf("parse invocations: got %d want %d (no extra parses when ctx.Parsed is set)", got, baseline)
	}
}

func TestSharedParseResultLazyFromContent(t *testing.T) {
	warmProjectStubIndex()
	syntax.ResetParseInvocationCount()

	src := []byte(sharedParseFixture)
	nodes, diags := syntax.ParseAST(src)
	if len(diags) > 0 {
		t.Fatalf("unexpected parse diagnostics: %v", diags)
	}
	baseline := syntax.ParseInvocationCount()
	if baseline == 0 {
		t.Fatal("expected baseline parse count > 0 after syntax.ParseAST")
	}

	level := 8
	project := BuildProjectIndex(map[string][]ast.Node{"t.php": nodes})
	ctx := &AnalysisContext{
		Content:       src,
		AnalysisLevel: &level,
		Resolver:      project,
	}
	_ = RunAnalysisRulesWithContext("t.php", nodes, ctx)

	want := baseline + 1
	if got := syntax.ParseInvocationCount(); got != want {
		t.Fatalf("parse invocations: got %d want %d (exactly one shared re-parse for CST rules)", got, want)
	}
	if ctx.Parsed == nil {
		t.Fatal("expected RunAnalysisRulesWithContext to populate ctx.Parsed")
	}
}
