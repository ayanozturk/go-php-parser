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
	if level0Report.Totals.Cases != 94 {
		t.Fatalf("expected 94 level-0 differential cases, got %d", level0Report.Totals.Cases)
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

	level2Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential-level2"), "", true)
	if err != nil {
		t.Fatalf("run engine level-2 differential baseline: %v", err)
	}
	if level2Report.Totals.Cases != 96 || level2Report.Totals.EngineMismatches != 0 {
		t.Fatalf("unexpected level-2 differential baseline: %#v", level2Report.Totals)
	}
	if level2Report.Reference != nil {
		t.Fatalf("engine-only level-2 report should not contain a reference run: %#v", level2Report.Reference)
	}

	level3Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential-level3"), "", true)
	if err != nil {
		t.Fatalf("run engine level-3 differential baseline: %v", err)
	}
	if level3Report.Totals.Cases != 30 || level3Report.Totals.EngineMismatches != 0 {
		t.Fatalf("unexpected level-3 differential baseline: %#v", level3Report.Totals)
	}
	if level3Report.Reference != nil {
		t.Fatalf("engine-only level-3 report should not contain a reference run: %#v", level3Report.Reference)
	}

	level5Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential-level5"), "", true)
	if err != nil {
		t.Fatalf("run engine level-5 differential baseline: %v", err)
	}
	if level5Report.Totals.Cases != 6 || level5Report.Totals.EngineMismatches != 0 {
		t.Fatalf("unexpected level-5 differential baseline: %#v", level5Report.Totals)
	}
	if level5Report.Reference != nil {
		t.Fatalf("engine-only level-5 report should not contain a reference run: %#v", level5Report.Reference)
	}

	level6Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential-level6"), "", true)
	if err != nil {
		t.Fatalf("run engine level-6 differential baseline: %v", err)
	}
	if level6Report.Totals.Cases != 16 || level6Report.Totals.EngineMismatches != 0 {
		t.Fatalf("unexpected level-6 differential baseline: %#v", level6Report.Totals)
	}
	if level6Report.Reference != nil {
		t.Fatalf("engine-only level-6 report should not contain a reference run: %#v", level6Report.Reference)
	}

	level7Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential-level7"), "", true)
	if err != nil {
		t.Fatalf("run engine level-7 differential baseline: %v", err)
	}
	if level7Report.Totals.Cases != 5 || level7Report.Totals.EngineMismatches != 0 {
		t.Fatalf("unexpected level-7 differential baseline: %#v", level7Report.Totals)
	}
	if level7Report.Reference != nil {
		t.Fatalf("engine-only level-7 report should not contain a reference run: %#v", level7Report.Reference)
	}

	level8Report, err := runDifferential(filepath.Join("..", "..", "testdata", "diagnostic-differential-level8"), "", true)
	if err != nil {
		t.Fatalf("run engine level-8 differential baseline: %v", err)
	}
	if level8Report.Totals.Cases != 5 || level8Report.Totals.EngineMismatches != 0 {
		t.Fatalf("unexpected level-8 differential baseline: %#v", level8Report.Totals)
	}
	if level8Report.Reference != nil {
		t.Fatalf("engine-only level-8 report should not contain a reference run: %#v", level8Report.Reference)
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
