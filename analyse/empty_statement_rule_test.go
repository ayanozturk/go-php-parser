package analyse

import (
	"testing"
)

func runEmptyStatementAnalysis(t *testing.T, php string) []AnalysisIssue {
	t.Helper()
	r := &EmptyStatementRule{}
	return r.CheckIssuesWithSource("test.php", []byte(php), nil)
}

func countEmptyStatementIssues(issues []AnalysisIssue) int {
	cnt := 0
	for _, iss := range issues {
		if iss.Code == "Generic.CodeAnalysis.EmptyStatement" {
			cnt++
		}
	}
	return cnt
}

func hasEmptyStatementIssue(issues []AnalysisIssue) bool {
	return countEmptyStatementIssues(issues) > 0
}

func TestStandaloneSemicolon(t *testing.T) {
	php := "<?php\n;\n$z = 1;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if countEmptyStatementIssues(issues) != 1 {
		t.Fatalf("expected 1 empty statement, got %d (issues=%v)", countEmptyStatementIssues(issues), issues)
	}
}

func TestIfWithSemicolonBody(t *testing.T) {
	php := "<?php\nif ($x) ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty statement for if(...);, got %v", issues)
	}
}

func TestWhileWithSemicolonBody(t *testing.T) {
	php := "<?php\nwhile ($x) ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty statement for while(...);, got %v", issues)
	}
}

func TestForWithSemicolonBody(t *testing.T) {
	php := "<?php\nfor($i=0;$i<10;$i++) ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if !hasEmptyStatementIssue(issues) {
		t.Fatalf("expected empty statement for for(...);, got %v", issues)
	}
}

func TestForHeaderSemicolonsIgnored(t *testing.T) {
	php := "<?php\nfor($i=0;$i<10;$i++) { }\n"
	issues := runEmptyStatementAnalysis(t, php)
	if hasEmptyStatementIssue(issues) {
		t.Fatalf("did not expect empty statement for for header semicolons, got %v", issues)
	}
}

func TestMultipleEmptySemicolons(t *testing.T) {
	php := "<?php\n; ; ;\n"
	issues := runEmptyStatementAnalysis(t, php)
	if countEmptyStatementIssues(issues) != 3 {
		t.Fatalf("expected 3 empty statements, got %d (%v)", countEmptyStatementIssues(issues), issues)
	}
}

func TestNoFalsePositiveInStatements(t *testing.T) {
	php := "<?php\n$a = 1;\n$b = 2;\nif ($a) { echo $b; }\n"
	issues := runEmptyStatementAnalysis(t, php)
	if hasEmptyStatementIssue(issues) {
		t.Fatalf("unexpected empty statement issue: %v", issues)
	}
}
