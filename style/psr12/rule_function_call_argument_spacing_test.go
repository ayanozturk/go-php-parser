package psr12

import "testing"

func TestFunctionCallArgumentSpacingChecker(t *testing.T) {
	checker := &FunctionCallArgumentSpacingChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		// OK: correct spacing
		{[]string{"foo(1, 2, 3);"}, 0, "correct spacing"},
		// Bad: multiple spaces after comma
		{[]string{"foo(1,  2, 3);"}, 1, "multiple spaces after comma"},
		// Bad: space before comma
		{[]string{"foo(1 ,2, 3);"}, 1, "space before comma"},
		// Bad: no space after comma
		{[]string{"foo(1,2, 3);"}, 1, "no space after comma"},
		// Bad: multiple errors
		{[]string{"foo( 1,2 ,  3 );"}, 1, "multiple errors in one call"},
		// OK: single argument
		{[]string{"foo($x);"}, 0, "single argument"},
		// OK: nested call
		{[]string{"foo(bar(1, 2), 3);"}, 0, "nested call with correct spacing"},
	}

	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}
