package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
)

func TestReadmeRuleLevelTableMatchesRegistry(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}

	want := renderRuleLevelTable(analyse.ListRegisteredAnalysisRuleMetadata())
	text := string(readme)
	start := strings.Index(text, tableStart)
	end := strings.Index(text, tableEnd)
	if start < 0 || end < start {
		t.Fatalf("README must contain %s and %s markers", tableStart, tableEnd)
	}
	end += len(tableEnd)
	if got := text[start:end]; got != want {
		t.Fatalf("README analysis-rule table is stale; replace it with:\n%s", want)
	}
}

func TestLevelDetailMetadataMatchesRegistry(t *testing.T) {
	introduced := make([]int, maxPHPStanLevel+1)
	unlevelled := 0
	for _, rule := range analyse.ListRegisteredAnalysisRuleMetadata() {
		if rule.Level < 0 || rule.Level > maxPHPStanLevel {
			unlevelled++
			continue
		}
		introduced[rule.Level]++
	}

	cumulative := 0
	for level, count := range introduced {
		cumulative += count
		path := filepath.Join("..", "..", "docs", "rules", fmt.Sprintf("level-%d.md", level))
		assertFileContains(t, path, fmt.Sprintf("<!-- rule-inventory: level=%d introduced=%d cumulative=%d -->", level, count, cumulative))
	}
	assertFileContains(t, "../../docs/rules/unlevelled.md", fmt.Sprintf(
		"<!-- rule-inventory: unlevelled=%d levelled=%d total=%d -->", unlevelled, cumulative, cumulative+unlevelled,
	))
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s inventory metadata is stale; expected %q", path, want)
	}
}
