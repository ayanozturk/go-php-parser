package analyse

import (
	"path/filepath"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestHybridOneParseOwnsFactsAndRules(t *testing.T) {
	warmProjectStubIndex()
	host := filepath.Join("src", "app.php")
	src := []byte(`<?php
function demo(int $a): void {
    strlen($a, 1);
    echo $undefined;
}
`)

	indexNodes, diags := syntax.ParseASTForIndex(src)
	if len(diags) > 0 {
		t.Fatalf("index parse: %v", diags)
	}
	fullNodes, fullDiags := syntax.ParseAST(src)
	if len(fullDiags) > 0 {
		t.Fatalf("full parse: %v", fullDiags)
	}

	// Baseline: R2.5c releasing snapshot + rules rebuild via Content.
	idx := BuildProjectIndex(map[string][]ast.Node{host: indexNodes})
	releasingParsed := map[string][]ast.Node{host: fullNodes}
	releasing, err := NewSemanticSnapshotWithIndexReleasingParsed(idx, releasingParsed, nil, nil)
	if err != nil {
		t.Fatalf("releasing snapshot: %v", err)
	}
	releasing.ReleaseVariableFlowAST()
	level := 5
	wantCtx := releasing.NewAnalysisContext()
	wantCtx.AnalysisLevel = &level
	wantCtx.Content = src
	want := RunAnalysisRulesWithContext(host, nil, wantCtx)

	// Hybrid: index-only snapshot; one ParseAndLower owns facts + rules.
	hybridIdx := BuildProjectIndex(map[string][]ast.Node{host: indexNodes})
	hybrid, err := NewSemanticSnapshotWithIndexOnly(hybridIdx, []string{host}, nil)
	if err != nil {
		t.Fatalf("index-only snapshot: %v", err)
	}
	nodes, res := syntax.ParseAndLower(src)
	gotCtx := hybrid.AnalysisContextForFile(host, nodes)
	gotCtx.AnalysisLevel = &level
	gotCtx.Content = src
	gotCtx.Parsed = res
	got := RunAnalysisRulesWithContext(host, nodes, gotCtx)

	if len(got) == 0 {
		t.Fatal("expected diagnostics from hybrid path")
	}
	if len(got) != len(want) {
		t.Fatalf("hybrid issue count %d != releasing %d\n got %#v\nwant %#v", len(got), len(want), got, want)
	}
}

func TestAnalysisContextForFileDoesNotPinLazyAST(t *testing.T) {
	host := filepath.Join("src", "app.php")
	src := []byte(`<?php
function demo($x) {
    echo $y;
}
`)
	nodes, diags := syntax.ParseAST(src)
	if len(diags) > 0 {
		t.Fatalf("parse: %v", diags)
	}
	idx := BuildProjectIndex(map[string][]ast.Node{host: nodes})
	snap, err := NewSemanticSnapshotWithIndexOnly(idx, []string{host}, nil)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	ctx := snap.AnalysisContextForFile(host, nodes)
	if ctx.VariableFlow == nil {
		t.Fatal("expected VariableFlow bound")
	}
	fileSnap, ok := ctx.Facts.(*SemanticSnapshot)
	if !ok {
		t.Fatalf("Facts type %T", ctx.Facts)
	}
	lazy := fileSnap.completeVariableReads[host]
	if lazy == nil {
		t.Fatal("expected lazy variable-flow entry")
	}
	if lazy.nodes != nil {
		t.Fatal("expected lazy AST pins cleared (R2.5c RSS model)")
	}
}
