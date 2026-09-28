package style

import "testing"

func TestNoBlankLineAfterPHPOpeningTagChecker(t *testing.T) {
	checker := &NoBlankLineAfterPHPOpeningTagChecker{}
	filename := "test.php"
	cases := []struct {
		lines    []string
		expected int
		msg      string
	}{
		{[]string{"<?php", "$a = 1;"}, 1, "missing blank line after opening"},
		{[]string{"<?php", "", "$a = 1;"}, 0, "blank line after opening"},
		{[]string{"<?php", "// comment"}, 1, "missing blank line before comment after opening"},
		{[]string{"<?php", "", "", "$a = 1;"}, 0, "multiple blank lines after opening (only first)"},
		{[]string{"<?php $a = 1;"}, 0, "code on same line as opening"},
		{[]string{"$a = 1;"}, 0, "no opening tag"},
	}
	for _, tc := range cases {
		issues := checker.CheckIssues(tc.lines, filename)
		if len(issues) != tc.expected {
			t.Errorf("%s: expected %d issues, got %d: %+v", tc.msg, tc.expected, len(issues), issues)
		}
		if tc.expected > 0 && tc.msg == "missing blank line after opening" {
			if issues[0].Line != 2 || issues[0].Column != 1 || issues[0].EndLine != 2 || issues[0].EndColumn != 1 {
				t.Errorf("expected insertion point at start of line 2, got %+v", issues[0])
			}
		}
	}
}
