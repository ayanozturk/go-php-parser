package analyse

import (
	"sort"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// astLanguageIssues runs the ast.Node-based language checks the same way
// Level0Rule.CheckIssues does internally (checkLanguageOnNode fed by
// walkAllWithFileContext, followed by goto/label resolution), without going
// through the full fused Level0Rule.CheckIssues walk (which also runs
// symbol/type/class-model checks that are irrelevant to this comparison).
func astLanguageIssues(t *testing.T, filename, src string) []AnalysisIssue {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(src))
	if len(diags) > 0 {
		t.Fatalf("unexpected parse diagnostics for %s: %v", filename, diags)
	}
	return (&Level0Rule{}).checkLanguage(filename, nodes, nil, FileTypeContext{})
}

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

func TestCheckLanguageIssuesFromCSTMatchesASTPath(t *testing.T) {
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

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			astIssues := sortIssuesForCompare(astLanguageIssues(t, filename, src))
			cstIssues := sortIssuesForCompare(CheckLanguageIssuesFromCST(filename, []byte(src)))

			if len(astIssues) != len(cstIssues) {
				t.Fatalf("issue count mismatch: ast=%d cst=%d\nast=%#v\ncst=%#v", len(astIssues), len(cstIssues), astIssues, cstIssues)
			}
			for i := range astIssues {
				a, c := astIssues[i], cstIssues[i]
				if a.Code != c.Code || a.Message != c.Message || a.Line != c.Line || a.Column != c.Column {
					t.Fatalf("issue %d mismatch:\nast=%#v\ncst=%#v", i, a, c)
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
