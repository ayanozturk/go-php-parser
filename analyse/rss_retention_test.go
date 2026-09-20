package analyse

import (
	"path/filepath"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestDropFileTypeASTRefsClearsClassNodes(t *testing.T) {
	t.Parallel()
	parsed := parseProjectSources(t, map[string]string{
		"src/App.php": `<?php
class App {
    public function run(): void {}
}
`,
	})
	idx := BuildProjectIndex(parsed)
	ft, ok := idx.FileTypes["src/App.php"]
	if !ok || len(ft.ClassNodes) == 0 {
		t.Fatal("expected ClassNodes after index build")
	}
	idx.DropSourceFiles()
	idx.DropFileTypeASTRefs()
	ft = idx.FileTypes["src/App.php"]
	if ft.ClassNodes != nil {
		t.Fatalf("ClassNodes still retained: %#v", ft.ClassNodes)
	}
	if _, ok := idx.ResolveClass("App"); !ok {
		t.Fatal("expected resolved class symbols to survive DropFileTypeASTRefs")
	}
}

func TestDropNonHostParsedTrees(t *testing.T) {
	t.Parallel()
	host := filepath.Join("src", "app.php")
	vendored := filepath.Join("vendor", "pkg", "Lib.php")
	hostNodes := []ast.Node{&ast.ExpressionStmt{}}
	vendorNodes := []ast.Node{&ast.ExpressionStmt{}}
	parsed := map[string][]ast.Node{
		host:     hostNodes,
		vendored: vendorNodes,
	}
	DropNonHostParsedTrees(parsed)
	if parsed[host] == nil {
		t.Fatal("host AST must remain")
	}
	if parsed[vendored] != nil {
		t.Fatal("vendored AST should be nilled after DropNonHostParsedTrees")
	}
}

func TestReleaseVariableFlowASTDropsLazyNodes(t *testing.T) {
	host := filepath.Join("src", "app.php")
	parsed := map[string][]ast.Node{
		host: parsePHPForProjectIndex(t, `<?php
function demo($x) {
    echo $y;
}
`),
	}
	snapshot, err := NewSemanticSnapshot(parsed, nil)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	lazy := snapshot.completeVariableReads[host]
	if lazy == nil || lazy.nodes == nil {
		t.Fatal("expected lazy variable-flow nodes after default snapshot build")
	}
	snapshot.ReleaseVariableFlowAST()
	if lazy.nodes != nil {
		t.Fatal("expected ReleaseVariableFlowAST to nil lazy nodes")
	}
	// Partial reads used by diagnostics must remain.
	var saw bool
	snapshot.rangeVariableReadsForFile(host, func(VariableReadFact) { saw = true })
	if !saw {
		t.Fatal("expected partial variable reads to remain after ReleaseVariableFlowAST")
	}
}

func TestGenerateScopedSemanticsReleasesHostAST(t *testing.T) {
	host := filepath.Join("src", "app.php")
	nodes := parsePHPForProjectIndex(t, `<?php
function demo($x) {
    echo $y;
}
`)
	parsed := map[string][]ast.Node{host: nodes}
	idx := BuildProjectIndex(parsed)
	snapshot, err := NewSemanticSnapshotWithIndexReleasingParsed(idx, parsed, nil, nil)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if parsed[host] != nil {
		t.Fatal("expected host AST to be released from parsed after releasing snapshot")
	}
	lazy := snapshot.completeVariableReads[host]
	if lazy == nil || lazy.nodes != nil {
		t.Fatal("expected lazy variable-flow AST pins to be cleared when releasing parsed")
	}
	if len(snapshot.Files()) != 1 || snapshot.Files()[0] != host {
		t.Fatalf("snapshot files = %#v", snapshot.Files())
	}
	var saw bool
	snapshot.rangeVariableReadsForFile(host, func(VariableReadFact) { saw = true })
	if !saw {
		t.Fatal("expected partial variable reads after releasing host AST")
	}
}

func TestRunAnalysisRulesLazyIngestFromContent(t *testing.T) {
	warmProjectStubIndex()
	src := []byte(`<?php
function demo(int $a): void {
    strlen($a, 1);
}
`)
	nodes, diags := syntax.ParseAST(src)
	if len(diags) > 0 {
		t.Fatalf("unexpected parse diagnostics: %v", diags)
	}
	project := BuildProjectIndex(map[string][]ast.Node{"t.php": nodes})
	// Keep a separate map so default snapshot does not matter; exercise lazy path.
	snapshot, err := NewSemanticSnapshotWithIndex(project, map[string][]ast.Node{"t.php": nodes}, nil, nil)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	level := 5
	withNodes := snapshot.NewAnalysisContext()
	withNodes.AnalysisLevel = &level
	withNodes.Content = src
	want := RunAnalysisRulesWithContext("t.php", nodes, withNodes)

	lazy := snapshot.NewAnalysisContext()
	lazy.AnalysisLevel = &level
	lazy.Content = src
	got := RunAnalysisRulesWithContext("t.php", nil, lazy)
	if len(got) == 0 {
		t.Fatal("expected diagnostics from lazy ingest ParseAndLower")
	}
	if len(got) != len(want) {
		t.Fatalf("lazy ingest issue count %d != retained nodes %d\n got %#v\nwant %#v", len(got), len(want), got, want)
	}
}
