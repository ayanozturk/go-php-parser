package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/overrides"
)

func TestFilterIssuesNilMatcherPassthrough(t *testing.T) {
	issues := []AnalysisIssue{
		{Code: "A.RULE", SubjectKind: "class", SubjectName: "Foo"},
	}
	got := FilterIssues(issues, nil)
	if len(got) != 1 || got[0].Code != "A.RULE" {
		t.Fatalf("nil matcher should return issues unchanged, got %#v", got)
	}
}

func TestFilterIssuesDropsMatchingClassSubjects(t *testing.T) {
	compiled, err := overrides.Compile(overrides.RuleOverrides{
		"A.RULE": {Classes: []string{"/^Legacy_/"}},
	})
	if err != nil {
		t.Fatalf("compile overrides: %v", err)
	}
	if compiled == nil {
		t.Fatal("expected non-nil compiled overrides")
	}

	issues := []AnalysisIssue{
		{Code: "A.RULE", SubjectKind: "class", SubjectName: "Legacy_Service"},
		{Code: "A.RULE", SubjectKind: "class", SubjectName: "ModernService"},
		{Code: "B.RULE", SubjectKind: "class", SubjectName: "Legacy_Service"},
		{Code: "A.RULE", SubjectKind: "function", SubjectName: "Legacy_Service"},
	}
	got := FilterIssues(issues, compiled)
	if len(got) != 3 {
		t.Fatalf("expected 3 remaining issues, got %d (%#v)", len(got), got)
	}
	for _, iss := range got {
		if iss.Code == "A.RULE" && iss.SubjectKind == "class" && iss.SubjectName == "Legacy_Service" {
			t.Fatalf("ignored class subject should have been filtered: %#v", iss)
		}
	}
}
