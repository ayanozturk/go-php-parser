package analyse

import (
	"path/filepath"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestIsVendoredPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want bool
	}{
		{"src/Plugin.php", false},
		{"vendor.php", false},
		{"vendor/pkg/src.php", true},
		{filepath.Join("app", "vendor", "foo.php"), true},
	}
	for _, tc := range cases {
		if got := IsVendoredPath(tc.path); got != tc.want {
			t.Errorf("IsVendoredPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSemanticSnapshotSkipsVendoredTypeChecking(t *testing.T) {
	host := filepath.Join("src", "app.php")
	vendored := filepath.Join("vendor", "pkg", "Lib.php")
	parsed := map[string][]ast.Node{
		host: parsePHPForProjectIndex(t, `<?php
function use_lib(): VendorLib {
    return new VendorLib();
}
`),
		vendored: parsePHPForProjectIndex(t, `<?php
class VendorLib {
    public function broken(): int {
        return "nope";
    }
}
function missing_vendor_call() {
    unknown_vendor_fn();
}
`),
	}

	snapshot, err := NewSemanticSnapshot(parsed, nil)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if got := snapshot.Files(); len(got) != 1 || got[0] != host {
		t.Fatalf("snapshot files = %#v, want only host %q", got, host)
	}
	if _, ok := snapshot.ResolveClass("VendorLib"); !ok {
		t.Fatal("expected vendored VendorLib to remain in the project index")
	}
	if reads := snapshot.VariableReadsForFile(vendored); len(reads) != 0 {
		t.Fatalf("vendored file should not receive variable-flow analysis, got %#v", reads)
	}
	if facts := snapshot.FactsForFile(vendored); len(facts) != 0 {
		t.Fatalf("vendored file should not receive generated semantic facts, got %#v", facts)
	}

	level := 0
	ctx := snapshot.NewAnalysisContext()
	ctx.AnalysisLevel = &level
	hostIssues := RunAnalysisRulesWithContext(host, parsed[host], ctx)
	for _, issue := range hostIssues {
		if issue.Code == "Level0.Symbols" {
			t.Fatalf("host file should resolve the vendored class, got %#v", hostIssues)
		}
	}
}
