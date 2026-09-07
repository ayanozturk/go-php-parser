package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScoreUsesExactLocationAndIdentifierCrosswalk(t *testing.T) {
	engine := []diagnostic{
		{Path: "src/a.php", Line: 10, Code: "Level1.Variables"},
		{Path: "src/a.php", Line: 20, Code: "Level1.Variables"},
		{Path: "src/a.php", Line: 30, Code: "Unknown.Code"},
	}
	reference := []diagnostic{
		{Path: "src/a.php", Line: 10, Identifier: "variable.undefined"},
		{Path: "src/a.php", Line: 21, Identifier: "variable.undefined"},
	}
	crosswalk := map[int]map[string]map[string]bool{1: {"Level1.Variables": {"variable.undefined": true}}}

	got := score(1, engine, reference, crosswalk)
	if got.ExactMatches != 1 || got.EngineOnly != 2 || got.PHPStanOnly != 1 || got.UnmappedEngine != 1 {
		t.Fatalf("unexpected buckets: %#v", got)
	}
	if got.PrecisionPct != 33.33 || got.RecallPct != 50 || got.CompatibilityPct != 40 {
		t.Fatalf("unexpected percentages: %#v", got)
	}
}

func TestScoreTreatsDuplicateDiagnosticsAsSeparate(t *testing.T) {
	engine := []diagnostic{{Path: "a.php", Line: 1, Code: "A"}, {Path: "a.php", Line: 1, Code: "A"}}
	reference := []diagnostic{{Path: "a.php", Line: 1, Identifier: "a"}}
	got := score(0, engine, reference, map[int]map[string]map[string]bool{0: {"A": {"a": true}}})
	if got.ExactMatches != 1 || got.EngineOnly != 1 || got.CompatibilityPct != 66.67 {
		t.Fatalf("duplicates must affect precision: %#v", got)
	}
}

func TestScoreEmptySetsAreFullyCompatible(t *testing.T) {
	got := score(0, nil, nil, nil)
	if got.CompatibilityPct != 100 || got.PrecisionPct != 100 || got.RecallPct != 100 {
		t.Fatalf("unexpected empty score: %#v", got)
	}
}

func TestScoreUsesMaximumCompatibleMatching(t *testing.T) {
	engine := []diagnostic{{Path: "a.php", Line: 1, Code: "Broad"}, {Path: "a.php", Line: 1, Code: "Narrow"}}
	reference := []diagnostic{{Path: "a.php", Line: 1, Identifier: "x"}, {Path: "a.php", Line: 1, Identifier: "y"}}
	crosswalk := map[int]map[string]map[string]bool{0: {
		"Broad":  {"x": true, "y": true},
		"Narrow": {"x": true},
	}}
	got := score(0, engine, reference, crosswalk)
	if got.ExactMatches != 2 || got.CompatibilityPct != 100 {
		t.Fatalf("expected maximum matching, got %#v", got)
	}
}

func TestScoreUsesOnlyMappingsReviewedAtOrBelowLevel(t *testing.T) {
	engine := []diagnostic{{Path: "a.php", Line: 1, Code: "A"}}
	reference := []diagnostic{{Path: "a.php", Line: 1, Identifier: "high"}}
	crosswalk := map[int]map[string]map[string]bool{8: {"A": {"high": true}}}
	if got := score(1, engine, reference, crosswalk); got.ExactMatches != 0 || got.UnmappedEngine != 1 {
		t.Fatalf("higher-level mapping leaked into level 1: %#v", got)
	}
	if got := score(8, engine, reference, crosswalk); got.ExactMatches != 1 {
		t.Fatalf("level 8 mapping was not used: %#v", got)
	}
}

func TestLoadCrosswalkBuildsAllowedPairs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	content := []byte(`{"schemaVersion":1,"reference":{"tool":"PHPStan","level":0},"cases":[{"engineCodes":["A"],"phpstanIdentifiers":["a","b"]}]}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	crosswalk, sources, err := loadCrosswalk(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].SHA256 == "" || !crosswalk[0]["A"]["a"] || !crosswalk[0]["A"]["b"] {
		t.Fatalf("unexpected crosswalk: %#v %#v", crosswalk, sources)
	}
}

func TestPHPStanAnalyzedFilesExtractsOnlyExistingPHPPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.php")
	if err := os.WriteFile(path, []byte("<?php"), 0o600); err != nil {
		t.Fatal(err)
	}
	files := phpstanAnalyzedFiles(dir, "intro\n"+path+"\nmissing.php\n")
	if len(files) != 1 || !files["a.php"] {
		t.Fatalf("unexpected analyzed files: %#v", files)
	}
}

func TestPHPStanJSONStartIgnoresBracesInDebugPrefix(t *testing.T) {
	output := []byte("/project/src/{generated}.php\n{\"totals\":{},\"files\":{},\"errors\":[]}")
	start := phpstanJSONStart(output)
	if start < 0 || string(output[start:start+10]) != "{\"totals\":" {
		t.Fatalf("unexpected JSON start %d", start)
	}
}

func TestParseLevelsRejectsInvalidAndDeduplicates(t *testing.T) {
	levels, err := parseLevels("8,1,1,0")
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 3 || levels[0] != 0 || levels[1] != 1 || levels[2] != 8 {
		t.Fatalf("unexpected levels: %#v", levels)
	}
	if _, err := parseLevels("11"); err == nil {
		t.Fatal("expected invalid level")
	}
}
