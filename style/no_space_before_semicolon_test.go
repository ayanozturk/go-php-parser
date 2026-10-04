package style

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/diag"
)

func TestNoSpaceBeforeSemicolonChecker(t *testing.T) {
	checker := &NoSpaceBeforeSemicolonChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		{[]string{"$a = 1;"}, 0, "correct, no space"},
		{[]string{"$a = 1 ;"}, 1, "incorrect, space before semicolon"},
		{[]string{"$a = 1  ;"}, 1, "incorrect, multiple spaces before semicolon"},
		{[]string{"$a = 1; $b = 2 ;"}, 1, "multiple statements, second incorrect"},
		{[]string{"// $a = 1 ;"}, 0, "comment, should not flag"},
		{[]string{"$a = 1; // comment"}, 0, "code then comment, correct"},
		{[]string{"$a = 1\t;"}, 1, "tab before semicolon"},
	}

	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}

func TestNoSpaceBeforeSemicolonHasExactWhitespaceSpan(t *testing.T) {
	source := []byte("$a = 1 ;")
	issues := (&NoSpaceBeforeSemicolonChecker{}).CheckIssues([]string{string(source)}, "test.php")
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %#v", issues)
	}
	issue := issues[0]
	if issue.Line != 1 || issue.Column != 7 || issue.EndLine != 1 || issue.EndColumn != 8 {
		t.Fatalf("expected the single whitespace byte range [7,8), got %+v", issue)
	}
	got := issue.AsDiagnostic(source, "tusk")
	if got.Span != (diag.ByteSpan{Start: 6, End: 7}) {
		t.Fatalf("expected exact byte span [6,7), got %+v", got.Span)
	}
}
