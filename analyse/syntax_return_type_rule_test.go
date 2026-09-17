package analyse

import (
	"testing"
)

func TestCheckReturnTypeIssuesFromCST(t *testing.T) {
	cases := map[string]string{
		"declaredVsActual": `<?php
function f(): int { return "x"; }
`,
		"correctReturn": `<?php
function f(): int { return 1; }
`,
		"voidReturnsValue": `<?php
function f(): void { return 1; }
`,
		"missingReturnPath": `<?php
function partial(bool $c): int { if ($c) { return 1; } }
`,
		"allPathsReturn": `<?php
function all(bool $c): int { if ($c) { return 1; } else { return 2; } }
`,
		"methodInClass": `<?php
class C { public function m(): int { return 1; } }
`,
		"closureMismatch": `<?php
$fn = function(): int { return "x"; };
`,
		"interfaceMethod": `<?php
interface I { public function run($x); }
`,
		"neverFallthrough": `<?php
function n(): never {}
`,
	}

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"declaredVsActual": {
			{Message: "Function f: return type mismatch, declared: int, actual: [string] at 2:1", Line: 2, Column: 1},
		},
		"correctReturn": {},
		"voidReturnsValue": {
			{Message: "Function f returns void without side effects", Line: 2, Column: 1},
			{Message: "Function f with return type void should not return a value", Line: 2, Column: 22},
		},
		"missingReturnPath": {
			{Message: "Function partial: declared return type int but not all paths return a value", Line: 2, Column: 1},
		},
		"allPathsReturn": {},
		"methodInClass":  {},
		"closureMismatch": {
			{Message: "Function : return type mismatch, declared: int, actual: [string] at 2:7", Line: 2, Column: 7},
		},
		"interfaceMethod": {},
		"neverFallthrough": {
			{Message: "Function n should always terminate", Line: 2, Column: 1},
		},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, _ := buildStructuralTestContext(t, filename, src)
			got := sortIssuesForCompare(CheckReturnTypeIssuesFromCST(filename, []byte(src), ctx))

			wantIssues := want[name]
			if len(wantIssues) != len(got) {
				t.Fatalf("issue count mismatch: want=%d got=%d\nwant=%+v\ngot=%+v", len(wantIssues), len(got), wantIssues, got)
			}
			for i := range wantIssues {
				if wantIssues[i].Line != got[i].Line || wantIssues[i].Column != got[i].Column || wantIssues[i].Message != got[i].Message {
					t.Fatalf("issue %d mismatch:\nwant=%+v\ngot=%+v", i, wantIssues[i], got[i])
				}
			}
		})
	}
}

func TestCheckReturnTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckReturnTypeIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}
