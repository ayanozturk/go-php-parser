package analyse

import (
	"testing"
)

func TestCheckThrowTypeIssuesFromCST(t *testing.T) {
	cases := map[string]string{
		"throwsInvalidClass": `<?php
class NotThrowable {}
function run(): void {
    throw new NotThrowable();
}
`,
		"throwsValidException": `<?php
function run(): void {
    throw new \RuntimeException("oops");
}
`,
		"throwsTrait": `<?php
trait T {}
function run(T $t): void {
    throw $t;
}
`,
		"throwsEnum": `<?php
enum E { case A; }
function run(): void {
    throw E::A;
}
`,
		"throwExprForm": `<?php
class NotThrowable {}
function run(?string $x): string {
    return $x ?? throw new NotThrowable();
}
`,
		"throwUnresolvedClass": `<?php
function run(): void {
    throw new MissingException();
}
`,
	}

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"throwsInvalidClass": {
			{Message: "Invalid type NotThrowable to throw.", Line: 4, Column: 5},
		},
		"throwsValidException": {},
		"throwsTrait":          {},
		"throwsEnum":           {},
		"throwExprForm": {
			{Message: "Invalid type NotThrowable to throw.", Line: 4, Column: 18},
		},
		"throwUnresolvedClass": {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, _ := buildStructuralTestContext(t, filename, src)
			got := sortIssuesForCompare(CheckThrowTypeIssuesFromCST(filename, []byte(src), ctx))

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

func TestCheckThrowTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckThrowTypeIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}
