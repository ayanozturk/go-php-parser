package style

import "testing"

func TestDisallowMultipleStatementsPointsToSecondStatement(t *testing.T) {
	line := "$é = 1; $b = 2;"
	issues := (&DisallowMultipleStatementsSniff{}).CheckIssues([]string{line}, "test.php")
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %#v", issues)
	}
	wantColumn := 9 // `$é = 1;` then a space, then the second `$`.
	if issues[0].Line != 1 || issues[0].Column != wantColumn || issues[0].EndLine != 1 || issues[0].EndColumn != wantColumn {
		t.Fatalf("expected zero-width point at the second statement, got %+v", issues[0])
	}
}

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
		{[]string{"for ($i = 0; $i < $length; $i += $limit) {"}, 0, "for loop header"},
		{[]string{"for ($i = 0; $i < $length; $i += $limit) { $r[] = mb_substr($value, $i, $limit, $charset); }"}, 0, "for loop with single body statement"},
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
