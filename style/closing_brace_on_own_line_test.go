package style

import (
	"go-phpcs/ast"
	"testing"
)

func TestClosingBraceOnOwnLineChecker(t *testing.T) {
	checker := &ClosingBraceOnOwnLineChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		// Correct: class closing brace on its own line, followed by blank or EOF
		{[]string{"class Foo", "{", "    $a = 1;", "}", ""}, 0, "class brace on own line, blank after"},
		// Incorrect: code after class closing brace on same line
		{[]string{"class Foo", "{", "    $a = 1;", "} $b = 2;"}, 1, "code after class closing brace on same line"},
		// Incorrect: code on line after class closing brace
		{[]string{"class Foo", "{", "    $a = 1;", "}", "$b = 2;"}, 1, "code after class closing brace on next line"},
		// Correct: function closing brace on same line as code (should not flag)
		{[]string{"function bar() {", "    return 1;", "} // comment"}, 0, "function brace with comment after (not class)"},
		// Correct: multiple closing braces, all on own line
		{[]string{"class Foo", "{", "    function bar() { return 1; }", "}", ""}, 0, "inner function brace on same line is allowed"},
		// Correct: closing brace with whitespace
		{[]string{"    }", ""}, 0, "brace with leading spaces, blank after"},
		// Incorrect: class closing brace with trailing code, then blank
		{[]string{"class Foo", "{", "    $a = 1;", "    } $x = 1;", ""}, 1, "class brace with trailing code, blank after"},
	}
	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}

func TestClosingBraceOnOwnLineInit(t *testing.T) {
	// This will trigger the init function and ensure the rule is registered
	issues := RunSelectedRules("test.php", []byte("}"), []ast.Node{}, []string{closingBraceOnOwnLineCode})
	if len(issues) == 0 {
		t.Errorf("Expected at least one issue for a lone closing brace, got none")
	}
}
