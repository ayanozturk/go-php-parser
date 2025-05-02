package psr12

import "testing"

func TestClassBraceOnOwnLineChecker(t *testing.T) {
	checker := &ClassBraceOnOwnLineChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		// Correct: brace on its own line
		{[]string{"class Foo", "{"}, 0, "class brace correct"},
		{[]string{"interface Bar", "{"}, 0, "interface brace correct"},
		{[]string{"trait Baz", "{"}, 0, "trait brace correct"},
		{[]string{"enum E", "{"}, 0, "enum brace correct"},
		// Incorrect: brace on same line
		{[]string{"class Foo {"}, 1, "class brace wrong line"},
		{[]string{"interface Bar {"}, 1, "interface brace wrong line"},
		{[]string{"trait Baz {"}, 1, "trait brace wrong line"},
		{[]string{"enum E {"}, 1, "enum brace wrong line"},
		// Incorrect: brace after whitespace
		{[]string{"class Foo", "   {"}, 1, "class brace with leading spaces"},
		// Correct: unrelated lines
		{[]string{"$a = 1;"}, 0, "not a class declaration"},
	}
	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}
