package style

import "testing"

func TestDisallowMultipleStatementsSniff(t *testing.T) {
	sniff := &DisallowMultipleStatementsSniff{}
	filename := "test.php"

	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		{[]string{"$a = 1;"}, 0, "single statement"},
		{[]string{"$a = 1; $b = 2;"}, 1, "multiple statements"},
		{[]string{"$a = 1; $b = 2; $c = 3;"}, 1, "three statements"},
		{[]string{"$a = 1; $b = 2; // comment"}, 1, "code before comment"},
		{[]string{"# comment", "$a = 1;"}, 0, "hash comment and code"},
		{[]string{""}, 0, "empty line"},
	}

	for _, tc := range cases {
		issues := sniff.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}
