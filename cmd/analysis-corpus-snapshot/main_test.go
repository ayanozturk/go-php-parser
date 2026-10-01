package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestBuildSnapshotMatchesEagerSemanticSnapshot(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"contract.php": `<?php
function acceptsInt(int $value): void {}
`,
		"caller.php": `<?php
acceptsInt('wrong');
`,
	}
	paths := make([]string, 0, len(files))
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	got, err := buildSnapshot(root, paths, 2, 5, 0, 0, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	parsed := make(map[string][]ast.Node, len(paths))
	contents := make(map[string][]byte, len(paths))
	parseResults := make(map[string]*syntax.ParseResult, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		nodes, result := syntax.ParseAndLower(content)
		if result == nil || len(result.Diagnostics) != 0 {
			t.Fatalf("fixture %s did not parse cleanly: %#v", path, result)
		}
		parsed[path] = nodes
		contents[path] = content
		parseResults[path] = result
	}
	eager, err := analyse.NewSemanticSnapshot(parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := make(map[string][]string, len(paths))
	level := 5
	for _, path := range eager.Files() {
		want[path] = []string{}
		ctx := eager.NewAnalysisContext()
		ctx.AnalysisLevel = &level
		ctx.Content = contents[path]
		ctx.Parsed = parseResults[path]
		issues := analyse.RunAnalysisRulesWithContext(path, parsed[path], ctx)
		for _, issue := range issues {
			want[path] = append(want[path], issue.Code+"|"+strconv.Itoa(issue.Line)+"|"+strconv.Itoa(issue.Column)+"|"+strconv.Itoa(issue.EndLine)+"|"+strconv.Itoa(issue.EndColumn)+"|"+issue.Message)
		}
		sort.Strings(want[path])
	}

	if !reflect.DeepEqual(got.Issues, want) {
		t.Fatalf("streaming issue set differs from eager snapshot:\n got: %#v\nwant: %#v", got.Issues, want)
	}
}

func TestBuildSnapshotAnalysisRangeReusesFullProjectIndex(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"a-caller.php": `<?php
acceptsInt('wrong');
`,
		"z-contract.php": `<?php
function acceptsInt(int $value): void {}
`,
	}
	paths := make([]string, 0, len(files))
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	got, err := buildSnapshot(root, paths, 1, 5, 0, 1, "", "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	caller := filepath.Join(root, "a-caller.php")
	if len(got.Issues) != 1 {
		t.Fatalf("target range returned %d issue files, want 1: %#v", len(got.Issues), got.Issues)
	}
	issues, ok := got.Issues[caller]
	if !ok {
		t.Fatalf("target range did not analyze first sorted file %s: %#v", caller, got.Issues)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "A.ARG.TYPE|") {
		t.Fatalf("target analysis did not resolve the declaration from the indexed non-target file: %#v", issues)
	}
}
