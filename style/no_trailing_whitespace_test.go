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
