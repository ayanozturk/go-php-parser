package analyse

import (
	"go-phpcs/ast"
	"sort"
	"testing"
)

func TestListRegisteredAnalysisRuleCodes(t *testing.T) {
	ClearAnalysisRules()

	RegisterAnalysisRule("Z.TEST.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue { return nil })
	RegisterAnalysisRule("A.TEST.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue { return nil })

	codes := ListRegisteredAnalysisRuleCodes()
	if len(codes) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(codes))
	}
	if !sort.StringsAreSorted(codes) {
		t.Errorf("codes are not sorted: %v", codes)
	}

	foundA, foundZ := false, false
	for _, c := range codes {
		if c == "A.TEST.RULE" {
			foundA = true
		}
		if c == "Z.TEST.RULE" {
			foundZ = true
		}
	}
	if !foundA || !foundZ {
		t.Errorf("expected both A.TEST.RULE and Z.TEST.RULE to be present, got %v", codes)
	}
}

func TestRunAnalysisRulesDeterministicOrder(t *testing.T) {
	ClearAnalysisRules()

	RegisterAnalysisRule("B.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue {
		return []AnalysisIssue{{Filename: filename, Code: "B.RULE", Message: "B"}}
	})
	RegisterAnalysisRule("A.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue {
		return []AnalysisIssue{{Filename: filename, Code: "A.RULE", Message: "A"}}
	})

	issues := RunAnalysisRules("test.php", nil)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Code != "A.RULE" || issues[0].Message != "A" {
		t.Errorf("expected first issue from A.RULE, got %#v", issues[0])
	}
	if issues[1].Code != "B.RULE" || issues[1].Message != "B" {
		t.Errorf("expected second issue from B.RULE, got %#v", issues[1])
	}
}

func TestClearAnalysisRules(t *testing.T) {
	ClearAnalysisRules()

	RegisterAnalysisRule("SOME.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue { return nil })
	if len(ListRegisteredAnalysisRuleCodes()) != 1 {
		t.Fatalf("expected 1 rule registered")
	}

	ClearAnalysisRules()
	if len(ListRegisteredAnalysisRuleCodes()) != 0 {
		t.Fatalf("expected registry to be empty after ClearAnalysisRules")
	}
}
