package analyse

import (
	"testing"
)

func TestCheckMissingTypeIssuesFromCST(t *testing.T) {
	cases := map[string]string{
		"missingParamType": `<?php
function run($x): void {}
`,
		"missingReturnType": `<?php
function run(int $x) {
    return $x;
}
`,
		"missingPropertyType": `<?php
class C {
    public $field;
}
`,
		"missingGenericType": `<?php
/**
 * @param array $x
 */
function run(array $x): void {}
`,
		"missingIterableValueType": `<?php
/**
 * @param iterable $x
 */
function run(iterable $x): void {}
`,
		"interfaceMethodMissingType": `<?php
interface I {
    public function run($x);
}
`,
		"constructorExemptFromReturnType": `<?php
class C {
    public function __construct(int $x) {}
}
`,
		"okAllTypesPresent": `<?php
class C {
    public int $field;
    public function run(int $x): int {
        return $x;
    }
}
`,
		"docParamTypeSuppresses": `<?php
/**
 * @param int $x
 */
function run($x): void {}
`,
	}

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"missingParamType": {
			{Message: "Parameter $x has no type specified.", Line: 2, Column: 14},
		},
		"missingReturnType": {
			{Message: "Function or method run has no return type specified.", Line: 2, Column: 1},
		},
		"missingPropertyType": {
			{Message: "Property $field has no type specified.", Line: 3, Column: 5},
		},
		"missingGenericType": {
			{Message: "Iterable type array does not specify its value type.", Line: 5, Column: 14},
		},
		"missingIterableValueType": {
			{Message: "Iterable type iterable does not specify its value type.", Line: 5, Column: 14},
		},
		"interfaceMethodMissingType": {
			{Message: "Function or method run has no return type specified.", Line: 3, Column: 12},
			{Message: "Parameter $x has no type specified.", Line: 3, Column: 25},
		},
		"constructorExemptFromReturnType": {},
		"okAllTypesPresent":               {},
		"docParamTypeSuppresses":          {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, _ := buildStructuralTestContext(t, filename, src)
			got := sortIssuesForCompare(CheckMissingTypeIssuesFromCST(filename, []byte(src), ctx))

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

func TestCheckMissingTypeIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckMissingTypeIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}
