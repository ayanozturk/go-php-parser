package analyse

import (
	"testing"
)

func TestDependencyGraphFilesAffectedByChange(t *testing.T) {
	dg := NewDependencyGraph()
	hash := FileChecksum([]byte("base"))

	dg.AddFile("base.php", hash, SymbolSet{
		Classes: map[string]struct{}{"Base": {}},
	}, SymbolSet{})
	dg.AddFile("user.php", FileChecksum([]byte("user")), SymbolSet{
		Classes: map[string]struct{}{"User": {}},
	}, SymbolSet{
		Classes: map[string]struct{}{"Base": {}},
	})
	dg.AddFile("other.php", FileChecksum([]byte("other")), SymbolSet{
		Classes: map[string]struct{}{"Other": {}},
	}, SymbolSet{})

	affected := dg.FilesAffectedByChange("base.php")
	if !containsPath(affected, "base.php") || !containsPath(affected, "user.php") {
		t.Fatalf("expected base+user affected, got %v", affected)
	}
	if containsPath(affected, "other.php") {
		t.Fatalf("unrelated file should not be affected, got %v", affected)
	}

	onlySelf := dg.FilesAffectedByChange("missing.php")
	if len(onlySelf) != 1 || onlySelf[0] != "missing.php" {
		t.Fatalf("unknown file should only affect itself, got %v", onlySelf)
	}
}

func TestDependencyGraphAddFileReplacesReverseIndices(t *testing.T) {
	dg := NewDependencyGraph()
	hash := FileChecksum([]byte("v1"))

	dg.AddFile("lib.php", hash, SymbolSet{
		Classes: map[string]struct{}{"Old": {}},
	}, SymbolSet{
		Classes: map[string]struct{}{"Dep": {}},
	})
	dg.AddFile("lib.php", FileChecksum([]byte("v2")), SymbolSet{
		Classes: map[string]struct{}{"New": {}},
	}, SymbolSet{})

	if _, ok := dg.ReverseClassDeps["Old"]; ok {
		t.Fatalf("stale defined class should be removed from reverse deps: %#v", dg.ReverseClassDeps)
	}
	if paths := dg.ReverseClassDeps["New"]; len(paths) != 1 || paths[0] != "lib.php" {
		t.Fatalf("expected single New reverse dep, got %#v", dg.ReverseClassDeps["New"])
	}
	if _, ok := dg.ReverseClassUsage["Dep"]; ok {
		t.Fatalf("stale used class should be removed from reverse usage: %#v", dg.ReverseClassUsage)
	}

	// Second replace of the same definition must not duplicate reverse entries.
	dg.AddFile("lib.php", FileChecksum([]byte("v3")), SymbolSet{
		Classes: map[string]struct{}{"New": {}},
	}, SymbolSet{})
	if paths := dg.ReverseClassDeps["New"]; len(paths) != 1 || paths[0] != "lib.php" {
		t.Fatalf("re-add should not duplicate reverse deps, got %#v", paths)
	}
}

func TestFilesChangedDetectsAddsModifiesAndDeletes(t *testing.T) {
	old := map[string]string{"a.php": "1", "b.php": "2", "c.php": "3"}
	newChecksums := map[string]string{"a.php": "1", "b.php": "changed", "d.php": "4"}
	got := FilesChanged(newChecksums, old)
	want := map[string]bool{"b.php": true, "c.php": true, "d.php": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v", got, want)
	}
	for _, path := range got {
		if !want[path] {
			t.Fatalf("unexpected path %q in %v", path, got)
		}
	}
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
