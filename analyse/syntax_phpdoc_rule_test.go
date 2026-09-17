package analyse

import (
	"testing"
)

func TestCheckPHPDocIssuesFromCST(t *testing.T) {
	cases := map[string]string{
		"paramTypeMismatch": `<?php
class C {
    /**
     * @param int $x
     */
    public function run(string $x): void {}
}
`,
		"paramUnknownName": `<?php
class C {
    /**
     * @param int $missing
     */
    public function run(string $x): void {}
}
`,
		"returnTypeMismatch": `<?php
/**
 * @return int
 */
function run(): string {
    return "x";
}
`,
		"propertyTypeMismatch": `<?php
class C {
    /**
     * @var int
     */
    public string $field;
}
`,
		"interfaceMethodTypeMismatch": `<?php
interface I {
    /**
     * @param int $x
     */
    public function run(string $x): void;
}
`,
		"closureParamTypeMismatch": `<?php
function outer(): void {
    /**
     * @param int $x
     */
    $fn = function (string $x): void {};
}
`,
		"templateNameSuppressesMismatch": `<?php
/**
 * @template T
 * @param T $x
 */
function run(string $x): void {}
`,
		"okNoIssues": `<?php
class C {
    /**
     * @param int $x
     * @return int
     */
    public function run(int $x): int {
        return $x;
    }
}
`,
		// Regression: an anonymous class's property nested behind
		// `(new class {...})::class` has its class-part expression
		// discarded by splitStaticMemberAccessParts (only .TokenLiteral()
		// is kept - see syntax_walk.go's staticMemberAccessDynamicClassPart
		// doc comment), so the anonymous class's members (including this
		// property's PHPDoc) are entirely invisible to walkAllConfigured.
		// self::* is a legitimate PHPStan class-constant wildcard, not an
		// unknown class reference - but it's moot here since the property
		// should never be reached by either path.
		"anonClassBehindStaticClassConstUnreachable": `<?php
function run() {
    useIt((new class {
        /** @var self::*|null */
        public $foo;
    })::class);
}
`,
	}

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"paramTypeMismatch": {
			{Message: "PHPDoc type int for parameter $x is not compatible with native type string.", Line: 6, Column: 25},
		},
		"paramUnknownName": {
			{Message: "PHPDoc tag @param references unknown parameter $missing.", Line: 6, Column: 12},
		},
		"returnTypeMismatch": {
			{Message: "PHPDoc return type int is not compatible with native return type string.", Line: 5, Column: 1},
		},
		"propertyTypeMismatch": {
			{Message: "PHPDoc type int for property $field is not compatible with native type string.", Line: 6, Column: 5},
		},
		"interfaceMethodTypeMismatch": {
			{Message: "PHPDoc type int for parameter $x is not compatible with native type string.", Line: 6, Column: 25},
		},
		"closureParamTypeMismatch":                   {},
		"templateNameSuppressesMismatch":             {},
		"okNoIssues":                                 {},
		"anonClassBehindStaticClassConstUnreachable": {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, nodes := buildStructuralTestContext(t, filename, src)
			aliases := collectPHPDocTypeAliases(nodes)
			got := sortIssuesForCompare(CheckPHPDocIssuesFromCST(filename, []byte(src), ctx, aliases))

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

func TestCheckPHPDocIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckPHPDocIssuesFromCST("empty.php", nil, &AnalysisContext{}, nil); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

func TestCheckPHPDocIssuesFromCSTNilContext(t *testing.T) {
	if got := CheckPHPDocIssuesFromCST("empty.php", []byte("<?php\n"), nil, nil); got != nil {
		t.Fatalf("expected nil issues for nil ctx, got %+v", got)
	}
}
