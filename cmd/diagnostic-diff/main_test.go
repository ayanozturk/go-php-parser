package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInEngineDifferentialBaseline(t *testing.T) {
	level0Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential"), "", true)
	if err != nil {
		t.Fatalf("run engine differential baseline: %v", err)
	}
	if level0Report.Totals.Cases != 16 {
		t.Fatalf("expected 16 level-0 differential cases, got %d", level0Report.Totals.Cases)
	}
	if level0Report.Totals.EngineMismatches != 0 {
		t.Fatalf("engine differential baseline has %d mismatches: %#v", level0Report.Totals.EngineMismatches, level0Report.Cases)
	}
	if level0Report.Reference != nil {
		t.Fatalf("engine-only report should not contain a reference run: %#v", level0Report.Reference)
	}

	level1Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential-level1"), "", true)
	if err != nil {
		t.Fatalf("run engine level-1 differential baseline: %v", err)
	}
	if level1Report.Totals.Cases != 24 {
		t.Fatalf("expected 24 level-1 differential cases, got %d", level1Report.Totals.Cases)
	}
	if level1Report.Totals.EngineMismatches != 0 {
		t.Fatalf("engine level-1 differential baseline has %d mismatches: %#v", level1Report.Totals.EngineMismatches, level1Report.Cases)
	}
	if level1Report.Reference != nil {
		t.Fatalf("engine-only level-1 report should not contain a reference run: %#v", level1Report.Reference)
	}
}

func TestLoadManifestRejectsDuplicateCaseIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	content := []byte(`{
  "schemaVersion": 1,
  "reference": {"tool": "PHPStan", "level": 0},
  "cases": [
    {"id": "same", "capability": "one", "file": "one.php", "engineCodes": [], "phpstanIdentifiers": []},
    {"id": "same", "capability": "two", "file": "two.php", "engineCodes": [], "phpstanIdentifiers": []}
  ]
}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := loadManifest(path); err == nil {
		t.Fatal("expected duplicate case id to be rejected")
	}
}

func TestDecodePHPStanIdentifiers(t *testing.T) {
	identifiers, err := decodePHPStanIdentifiers([]byte(`{
  "files": {
    "/fixture.php": {
      "messages": [
        {"identifier": "variable.undefined"},
        {"identifier": "class.notFound"}
      ]
    }
  },
  "errors": []
}`))
	if err != nil {
		t.Fatalf("decode PHPStan output: %v", err)
	}
	if !equalStrings(identifiers, []string{"class.notFound", "variable.undefined"}) {
		t.Fatalf("unexpected identifiers: %#v", identifiers)
	}
}
