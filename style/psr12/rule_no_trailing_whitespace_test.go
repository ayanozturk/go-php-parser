package psr12

import (
	"testing"
)

func TestNoTrailingWhitespaceChecker(t *testing.T) {
	checker := &NoTrailingWhitespaceChecker{}
	filename := "test.php"
	lines := []string{
		"<?php",
		"class Foo { ", // trailing space
		"    public function bar() {\t", // trailing tab
		"        return 42;",
		"    }",
		"}",
	}
	errors := checker.Check(lines, filename)
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errors), errors)
	}
	if len(errors) > 0 && errors[0] != "[PSR12:NoTrailingWhitespace] File: test.php | Line: 2 | Error: Trailing whitespace detected" {
		t.Errorf("unexpected error message: %s", errors[0])
	}
	if len(errors) > 1 && errors[1] != "[PSR12:NoTrailingWhitespace] File: test.php | Line: 3 | Error: Trailing whitespace detected" {
		t.Errorf("unexpected error message: %s", errors[1])
	}
}
