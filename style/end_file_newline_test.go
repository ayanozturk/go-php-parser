package style

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/diag"
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
	} else if issues[0].Line != 2 || issues[0].Column != len(fooClass)+1 || issues[0].EndLine != 2 || issues[0].EndColumn != len(fooClass)+1 {
		t.Errorf("expected zero-width EOF insertion point on line 2, got %+v", issues[0])
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

func TestEndFileNewlineDiagnosticMapsToByteEOF(t *testing.T) {
	source := []byte("<?php\n$🙂 = 1;")
	issues := RunSelectedRules("test.php", source, nil, []string{endFileNewlineCode})
	if len(issues) != 1 {
		t.Fatalf("expected one missing-newline issue, got %#v", issues)
	}
	got := issues[0].AsDiagnostic(source, "go-php-parser")
	if got.Span != (diag.ByteSpan{Start: len(source), End: len(source)}) {
		t.Fatalf("expected zero-width byte span at EOF %d, got %+v", len(source), got.Span)
	}
}
