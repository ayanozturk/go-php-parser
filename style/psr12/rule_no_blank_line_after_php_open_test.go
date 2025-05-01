package psr12

import "testing"

func TestNoBlankLineAfterPHPOpeningTagChecker(t *testing.T) {
	checker := &NoBlankLineAfterPHPOpeningTagChecker{}
	filename := "test.php"
	cases := []struct {
		lines   []string
		expected int
		msg      string
	}{
		{[]string{"<?php", "$a = 1;"}, 0, "no blank line after opening"},
		{[]string{"<?php", "", "$a = 1;"}, 1, "blank line after opening"},
		{[]string{"<?php", "// comment"}, 0, "comment after opening"},
		{[]string{"<?php", "", "", "$a = 1;"}, 1, "multiple blank lines after opening (only first)"},
		{[]string{"<?php $a = 1;"}, 0, "code on same line as opening"},
		{[]string{"$a = 1;"}, 0, "no opening tag"},
	}
	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
	}
}
