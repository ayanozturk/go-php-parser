package style

import (
	"testing"
)

func TestEndFileNewlineChecker(t *testing.T) {
	const fooClass = "class Foo {}"
	checker := &EndFileNewlineChecker{}
	filename := "test.php"

	// Case 1: File ends with a single blank line (correct)
	lines := []string{"<?php", fooClass, ""}
	issues := checker.CheckIssues(lines, filename)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d: %+v", len(issues), issues)
	}

	// Case 2: File does not end with a blank line (incorrect)
	lines = []string{"<?php", fooClass}
	issues = checker.CheckIssues(lines, filename)
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d: %+v", len(issues), issues)
	} else if issues[0].Code != "PSR12.Files.EndFileNewline" {
		t.Errorf("expected code PSR12.Files.EndFileNewline, got %s", issues[0].Code)
	}

	// Case 3: File ends with multiple blank lines (incorrect)
	lines = []string{"<?php", fooClass, "", ""}
	issues = checker.CheckIssues(lines, filename)
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d: %+v", len(issues), issues)
	}

	// Case 4: Empty file (should not error)
	lines = []string{}
	issues = checker.CheckIssues(lines, filename)
	if len(issues) != 0 {
		t.Errorf("expected no issues for empty file, got %d: %+v", len(issues), issues)
	}
}
