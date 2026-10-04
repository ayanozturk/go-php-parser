package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/style"
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

func TestEveryRegisteredRuleHasDocumentedEntryExampleAndRationale(t *testing.T) {
	levelPages, err := filepath.Glob("../../docs/rules/level-*.md")
	if err != nil {
		t.Fatalf("list level rule pages: %v", err)
	}
	pages := append([]string(nil), levelPages...)
	pages = append(pages, "../../docs/rules/unlevelled.md", "../../docs/rules/style.md")
	docs := make(map[string]string, len(pages))
	for _, path := range pages {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		docs[path] = string(content)
	}

	want := make(map[string]bool)
	expectedPage := make(map[string]string)
	for _, rule := range analyse.ListRegisteredAnalysisRuleMetadata() {
		want[rule.Code] = true
		if rule.Level < 0 || rule.Level > maxPHPStanLevel {
			expectedPage[rule.Code] = "../../docs/rules/unlevelled.md"
		} else {
			expectedPage[rule.Code] = filepath.Join("..", "..", "docs", "rules", fmt.Sprintf("level-%d.md", rule.Level))
		}
	}
	for _, code := range style.ListRegisteredRuleCodes() {
		want[code] = true
		expectedPage[code] = "../../docs/rules/style.md"
	}

	entryHeading := regexp.MustCompile("(?m)^#{2,3} `([^`]+)`$")
	got := make(map[string]int)
	gotPage := make(map[string]string)
	for path, doc := range docs {
		matches := entryHeading.FindAllStringSubmatchIndex(doc, -1)
		for i, match := range matches {
			code := doc[match[2]:match[3]]
			got[code]++
			gotPage[code] = path
			end := len(doc)
			if i+1 < len(matches) {
				end = matches[i+1][0]
			}
			section := doc[match[1]:end]
			for _, required := range []string{"**What it checks:**", "**Why it helps:**", "**Example that reports:**"} {
				if !strings.Contains(section, required) {
					t.Errorf("%s entry %s is missing %s", path, code, required)
				}
			}
		}
	}
	for code := range want {
		if got[code] != 1 {
			t.Errorf("registered rule %s must have exactly one documented entry; found %d", code, got[code])
		}
		if gotPage[code] != expectedPage[code] {
			t.Errorf("registered rule %s must be documented in %s; found %s", code, expectedPage[code], gotPage[code])
		}
	}
	for code := range got {
		if !want[code] {
			t.Errorf("documentation contains unregistered rule entry %s", code)
		}
	}
}

func TestRuleDocumentationIndexCountsMatchRegistry(t *testing.T) {
	content, err := os.ReadFile("../../docs/rules/README.md")
	if err != nil {
		t.Fatalf("read rule documentation index: %v", err)
	}
	index := string(content)
	levelCounts := make([]int, maxPHPStanLevel+1)
	unlevelled := 0
	for _, rule := range analyse.ListRegisteredAnalysisRuleMetadata() {
		if rule.Level < 0 || rule.Level > maxPHPStanLevel {
			unlevelled++
			continue
		}
		levelCounts[rule.Level]++
	}
	for level, count := range levelCounts {
		want := fmt.Sprintf("| [Level %d](level-%d.md) | %d |", level, level, count)
		if !strings.Contains(index, want) {
			t.Errorf("rule documentation index is missing current level count %q", want)
		}
	}
	if want := fmt.Sprintf("| [Unlevelled rules](unlevelled.md) | %d |", unlevelled); !strings.Contains(index, want) {
		t.Errorf("rule documentation index is missing current unlevelled count %q", want)
	}
	if want := fmt.Sprintf("documents all %d registered style rules", len(style.ListRegisteredRuleCodes())); !strings.Contains(index, want) {
		t.Errorf("rule documentation index is missing current style-rule count %q", want)
	}
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
