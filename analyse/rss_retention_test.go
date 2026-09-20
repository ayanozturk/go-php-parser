package analyse

import (
	"path/filepath"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
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
		t.Fatal("expected lazy variable-flow nodes after snapshot build")
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
