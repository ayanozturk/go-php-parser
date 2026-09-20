package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestDiscoverPHPFilesIndexesRootVendorAutomatically(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkFixture(t, root, "src/keep.php")
	writeBenchmarkFixture(t, root, "vendor/phpunit/Assert.php")

	files, err := discoverPHPFiles(root, []string{"src"}, nil)
	if err != nil {
		t.Fatalf("discover PHP files: %v", err)
	}
	want := []string{
		filepath.Join(root, "src", "keep.php"),
		filepath.Join(root, "vendor", "phpunit", "Assert.php"),
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("unexpected discovered files:\nwant: %#v\n got: %#v", want, files)
	}
}

func TestDiscoverPHPFilesUsesConfiguredPathsAndExcludes(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkFixture(t, root, "src/keep.php")
	writeBenchmarkFixture(t, root, "src/js/excluded.php")
	writeBenchmarkFixture(t, root, "tests/test.php")
	writeBenchmarkFixture(t, root, "vendor/dependency.php")
	writeBenchmarkFixture(t, root, "outside/ignored.php")

	files, err := discoverPHPFiles(root, []string{"src", "tests", "vendor", "src"}, []string{"src/js"})
	if err != nil {
		t.Fatalf("discover PHP files: %v", err)
	}
	want := []string{
		filepath.Join(root, "src", "keep.php"),
		filepath.Join(root, "tests", "test.php"),
		filepath.Join(root, "vendor", "dependency.php"),
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("unexpected discovered files:\nwant: %#v\n got: %#v", want, files)
	}
}

func TestDiscoverPHPFilesHonorsGlobExcludes(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkFixture(t, root, "packages/keep/src/File.php")
	writeBenchmarkFixture(t, root, "nested/packages/foo/vendor/autoload.php")
	writeBenchmarkFixture(t, root, "splitter/vendor/autoload.php")

	files, err := discoverPHPFiles(root, []string{"packages", "nested", "splitter"}, []string{"splitter/vendor", "*/packages/**/vendor/*"})
	if err != nil {
		t.Fatalf("discover PHP files: %v", err)
	}
	want := []string{filepath.Join(root, "packages", "keep", "src", "File.php")}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("unexpected discovered files:\nwant: %#v\n got: %#v", want, files)
	}
}

func TestParseBenchmarkPathsRejectsRootEscape(t *testing.T) {
	for _, input := range []string{"../outside", "src/../../outside", "/absolute"} {
		if _, err := parseBenchmarkPaths(input, false); err == nil {
			t.Fatalf("expected %q to be rejected", input)
		}
	}
}

func TestDiscoverPHPFilesRejectsMissingConfiguredPath(t *testing.T) {
	if _, err := discoverPHPFiles(t.TempDir(), []string{"vendor"}, nil); err == nil {
		t.Fatal("expected a missing configured path to fail discovery")
	}
}

func writeBenchmarkFixture(t *testing.T, root, relative string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("<?php\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func TestRunAnalysisSkipsVendoredFiles(t *testing.T) {
	hostPHP := `<?php
function use_lib(): VendorLib {
    return new VendorLib();
}
`
	vendorPHP := `<?php
class VendorLib {
    public function broken(): int {
        return "nope";
    }
}
`
	hostNodes, hostDiags := syntax.ParseAST([]byte(hostPHP))
	if len(hostDiags) > 0 {
		t.Fatalf("parse host: %v", hostDiags)
	}
	vendorNodes, vendorDiags := syntax.ParseAST([]byte(vendorPHP))
	if len(vendorDiags) > 0 {
		t.Fatalf("parse vendor: %v", vendorDiags)
	}
	hostPath := filepath.Join("src", "app.php")
	vendorPath := filepath.Join("vendor", "pkg", "Lib.php")
	parsed := map[string][]ast.Node{
		hostPath:   hostNodes,
		vendorPath: vendorNodes,
	}
	contents := map[string][]byte{
		hostPath:   []byte(hostPHP),
		vendorPath: []byte(vendorPHP),
	}
	project := analyse.BuildProjectIndex(parsed)
	level := 10
	if got := runAnalysis(parsed, contents, project, &level, 2); got != 0 {
		t.Fatalf("vendored type errors should not be counted, got %d diagnostics", got)
	}
}

func TestRunAnalysisUsesSharedSemanticSnapshot(t *testing.T) {
	php := `<?php
function identifier(): string {
    $value = 42;
    return $value;
}
`
	nodes, diags := syntax.ParseAST([]byte(php))
	if len(diags) > 0 {
		t.Fatalf("parse fixture: %v", diags)
	}
	parsed := map[string][]ast.Node{"file.php": nodes}
	contents := map[string][]byte{"file.php": []byte(php)}
	project := analyse.BuildProjectIndex(parsed)
	level := 10
	if got := runAnalysis(parsed, contents, project, &level, 2); got == 0 {
		t.Fatal("expected snapshot-backed return-type diagnostics")
	}
}
