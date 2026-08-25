package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInEngineDifferentialBaseline(t *testing.T) {
	report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential"), "", true)
	if err != nil {
		t.Fatalf("run engine differential baseline: %v", err)
	}
	if report.Totals.Cases != 5 {
		t.Fatalf("expected 5 differential cases, got %d", report.Totals.Cases)
	}
	if report.Totals.EngineMismatches != 0 {
		t.Fatalf("engine differential baseline has %d mismatches: %#v", report.Totals.EngineMismatches, report.Cases)
	}
	if report.Reference != nil {
		t.Fatalf("engine-only report should not contain a reference run: %#v", report.Reference)
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
