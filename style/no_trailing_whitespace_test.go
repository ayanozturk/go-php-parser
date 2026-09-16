package style

import (
	"testing"
)

func TestNoTrailingWhitespaceChecker(t *testing.T) {
	checker := &NoTrailingWhitespaceChecker{}
	filename := "test.php"
	lines := []string{
		"<?php",
		"class Foo { ",                  // trailing space
		"    public function bar() {\t", // trailing tab
		"        return 42;",
		"    }",
		"}",
	}
	issues := checker.CheckIssues(lines, filename)
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d: %+v", len(issues), issues)
	}
	if len(issues) > 0 && (issues[0].Line != 2 || issues[0].Message != "Trailing whitespace detected") {
		t.Errorf("unexpected issue: %+v", issues[0])
	}
	if len(issues) > 1 && (issues[1].Line != 3 || issues[1].Message != "Trailing whitespace detected") {
		t.Errorf("unexpected issue: %+v", issues[1])
	}
}

func TestNoTrailingWhitespaceChecker_Span(t *testing.T) {
	checker := &NoTrailingWhitespaceChecker{}

	t.Run("single trailing space", func(t *testing.T) {
		issues := checker.CheckIssues([]string{"class Foo { "}, "test.php")
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		issue := issues[0]
		if issue.Line != 1 || issue.Column != 12 || issue.EndLine != 1 || issue.EndColumn != 13 {
			t.Errorf("expected Line:1 Column:12 EndLine:1 EndColumn:13, got Line:%d Column:%d EndLine:%d EndColumn:%d",
				issue.Line, issue.Column, issue.EndLine, issue.EndColumn)
		}
	})

	t.Run("spaces and tab", func(t *testing.T) {
		issues := checker.CheckIssues([]string{"abc  \t"}, "test.php")
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		issue := issues[0]
		if issue.Line != 1 || issue.Column != 4 || issue.EndLine != 1 || issue.EndColumn != 7 {
			t.Errorf("expected Line:1 Column:4 EndLine:1 EndColumn:7, got Line:%d Column:%d EndLine:%d EndColumn:%d",
				issue.Line, issue.Column, issue.EndLine, issue.EndColumn)
		}
	})
}
