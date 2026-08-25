package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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
