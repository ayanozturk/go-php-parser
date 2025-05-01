package psr12

import "testing"

func TestNoSpaceBeforeSemicolonChecker(t *testing.T) {
	checker := &NoSpaceBeforeSemicolonChecker{}
	filename := "test.php"
	cases := []struct {
		lines   []string
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
