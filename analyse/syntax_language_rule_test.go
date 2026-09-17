package analyse

import (
	"sort"
	"testing"
)

func sortIssuesForCompare(issues []AnalysisIssue) []AnalysisIssue {
	out := make([]AnalysisIssue, len(issues))
	copy(out, issues)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		if out[i].Column != out[j].Column {
			return out[i].Column < out[j].Column
		}
		return out[i].Message < out[j].Message
	})
	return out
}

func TestCheckLanguageIssuesFromCST(t *testing.T) {
	cases := map[string]string{
		"gotoAndLabel": `<?php
goto nowhere;
here:
echo 1;
goto here;
`,
		"duplicateArrayKeys": `<?php
$a = ['x' => 1, 'x' => 2, 5 => 'y', 5 => 'z'];
`,
		"voidUnsetCasts": `<?php
$x = (void) 1;
$y = (unset) 1;
`,
		"incrementNonWritable": `<?php
function f() {
    1++;
    (2 + 3)--;
}
`,
		"invalidRegex": `<?php
preg_match('/(unclosed/', $subject);
`,
		"printfPlaceholderMismatch": `<?php
printf("%s and %s", "only one");
sprintf("%s", "fine");
`,
		"includeMissingFile": `<?php
include 'definitely-does-not-exist-123.php';
require_once 'also-missing-456.php';
`,
		"clean": `<?php
function add($a, $b) {
    return $a + $b;
}
echo add(1, 2);
`,
		"switchBodyIsInvisibleToLanguageChecks": `<?php
function f($x) {
    switch (true) {
        case preg_match('/[0-7_]++/', $x):
            $y = (void) 1;
            goto missing;
            break;
    }
}
`,
	}

	type wantIssue struct {
		Code    string
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"gotoAndLabel": {
			{Code: "Level0.Language", Message: "Goto to undefined label nowhere.", Line: 2, Column: 1},
		},
		"duplicateArrayKeys": {
			{Code: "Level0.Language", Message: `Array has "x" duplicate key.`, Line: 2, Column: 17},
			{Code: "Level0.Language", Message: "Array has 5 duplicate key.", Line: 2, Column: 37},
		},
		"voidUnsetCasts": {
			{Code: "Level0.Language", Message: "Cannot cast to void.", Line: 2, Column: 6},
			{Code: "Level0.Language", Message: "Cannot cast to unset.", Line: 3, Column: 6},
		},
		"incrementNonWritable": {
			{Code: "Level0.Language", Message: "Cannot use ++ on non-variable expression.", Line: 3, Column: 5},
			{Code: "Level0.Language", Message: "Cannot use -- on non-variable expression.", Line: 4, Column: 5},
		},
		"invalidRegex": {
			{Code: "Level0.Language", Message: "Regex pattern is invalid: error parsing regexp: missing closing ): `(unclosed`", Line: 2, Column: 1},
		},
		"printfPlaceholderMismatch": {
			{Code: "Level0.Invocation", Message: "Call to function printf contains 2 placeholders, 1 values given.", Line: 2, Column: 1},
		},
		"includeMissingFile": {
			{Code: "Level0.Language", Message: `Path in include() "definitely-does-not-exist-123.php" is not a file or it does not exist.`, Line: 2, Column: 1},
			{Code: "Level0.Language", Message: `Path in require_once() "also-missing-456.php" is not a file or it does not exist.`, Line: 3, Column: 1},
		},
		"clean":                                 {},
		"switchBodyIsInvisibleToLanguageChecks": {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			cstIssues := sortIssuesForCompare(CheckLanguageIssuesFromCST(filename, []byte(src)))

			wantIssues := want[name]
			if len(cstIssues) != len(wantIssues) {
				t.Fatalf("issue count mismatch: want=%d got=%d\nwant=%#v\ngot=%#v", len(wantIssues), len(cstIssues), wantIssues, cstIssues)
			}
			for i := range wantIssues {
				w, c := wantIssues[i], cstIssues[i]
				if w.Code != c.Code || w.Message != c.Message || w.Line != c.Line || w.Column != c.Column {
					t.Fatalf("issue %d mismatch:\nwant=%#v\ngot=%#v", i, w, c)
				}
			}
		})
	}
}

func TestCheckLanguageIssuesFromCSTNilRoot(t *testing.T) {
	if issues := CheckLanguageIssuesFromCST("empty.php", []byte("")); issues != nil {
		t.Fatalf("expected nil issues for empty content, got %#v", issues)
	}
}
