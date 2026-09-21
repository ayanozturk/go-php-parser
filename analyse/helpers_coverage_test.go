package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestSingleFileVariableFlowReadsForFile(t *testing.T) {
	php := `<?php
function run($known): void {
    echo $known;
    echo $unknown;
}
`
	nodes, diags := syntax.ParseAST([]byte(php))
	if len(diags) > 0 {
		t.Fatalf("parser diagnostics: %v", diags)
	}

	var visited int
	forEachVariableRead("test.php", nodes, nil, func(VariableReadFact) {
		visited++
	})
	if visited == 0 {
		t.Fatal("expected at least one variable read")
	}

	// Rebuild with an explicit single-file flow to exercise VariableReadsForFile.
	reads := buildVariableFlowFacts("test.php", nodes, false, nil)
	flow := singleFileVariableFlow{filename: "test.php", reads: reads}
	got := flow.VariableReadsForFile("test.php")
	if len(got) == 0 {
		t.Fatal("expected VariableReadsForFile hits")
	}
	if other := flow.VariableReadsForFile("other.php"); other != nil {
		t.Fatalf("mismatched filename should return nil, got %#v", other)
	}

	var ranged int
	flow.rangeVariableReadsForFile("test.php", func(VariableReadFact) { ranged++ })
	if ranged != len(reads) {
		t.Fatalf("range visited %d, want %d", ranged, len(reads))
	}
	flow.rangeVariableReadsForFile("other.php", func(VariableReadFact) {
		t.Fatal("should not visit other filename")
	})
}

func TestCloneBoolMapCopiesIndependently(t *testing.T) {
	in := map[string]bool{"a": true, "b": false}
	out := cloneBoolMap(in)
	out["a"] = false
	out["c"] = true
	if !in["a"] || in["c"] {
		t.Fatalf("cloneBoolMap must deep-copy, in=%#v out=%#v", in, out)
	}
}

func TestDefaultVisibilityAndParamsFromNodes(t *testing.T) {
	if got := defaultVisibility(""); got != "public" {
		t.Fatalf("empty visibility => public, got %q", got)
	}
	if got := defaultVisibility("protected"); got != "protected" {
		t.Fatalf("preserved visibility, got %q", got)
	}

	params := paramsFromNodes([]ast.Node{
		&ast.ParamNode{Name: "x", TypeHint: &ast.IdentifierNode{Value: "int"}},
		&ast.ParamNode{Name: "y"},
	}, FileTypeContext{})
	if len(params) != 2 || params[0].Name != "x" {
		t.Fatalf("unexpected params %#v", params)
	}
}
