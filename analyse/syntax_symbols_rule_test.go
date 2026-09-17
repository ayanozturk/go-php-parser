package analyse

import (
	"testing"
)

func TestCheckSymbolIssuesFromCST(t *testing.T) {
	cases := map[string]string{
		"unresolvedInstantiation": `<?php
function run(): void {
    new Missing();
}
`,
		"abstractInstantiation": `<?php
abstract class Base {}
function run(): void {
    new Base();
}
`,
		"interfaceInstantiation": `<?php
interface HasRun {}
function run(): void {
    new HasRun();
}
`,
		"unresolvedFunctionCall": `<?php
missingFunction();
`,
		"classicStaticCall": `<?php
class Real {
    public static function bar(): void {}
}
function run(): void {
    Real::bar();
    Real::missing();
    Missing::bar();
}
`,
		"dynamicStaticCall": `<?php
class Real {
    public static function bar(): void {}
}
function run(): void {
    $m = 'bar';
    Real::{$m}();
}
`,
		"instanceMethodCall": `<?php
class Real {
    public function bar(): void {}
}
function run(Real $r): void {
    $r->bar();
    $r->missing();
}
`,
		"thisMethodCall": `<?php
class Real {
    public function bar(): void {}
    public function run(): void {
        $this->bar();
        $this->missing();
    }
    public static function staticRun(): void {
        $this->bar();
    }
}
`,
		"classConstFetch": `<?php
class Real {
    const FIELD = 1;
}
function run(): void {
    echo Real::FIELD;
    echo Real::MISSING;
    echo Missing::FIELD;
}
`,
		"propertyFetch": `<?php
class Real {
    public int $field = 1;
}
function run(Real $r): void {
    echo $r->field;
    echo $r->missing;
}
`,
		"chainedMethodCalls": `<?php
class A {
    public function b(): A {
        return $this;
    }
}
function run(A $a): void {
    $a->b()->missing();
}
`,
		"chainedPropertyAccess": `<?php
class A {
    public ?A $b = null;
}
function run(A $a): void {
    echo $a->b->missing;
}
`,
		"arrowFunctionKeepsEnclosingCurrentFn": `<?php
class Real {
    public function bar(): void {}
    public static function staticRun(): void {
        $fn = fn() => $this->bar();
    }
}
`,
		"switchBodyGapPreserved": `<?php
function run($x): void {
    switch ($x) {
        case 1:
            new Missing();
            break;
    }
}
`,
		"yieldExpressionGapPreserved": `<?php
function run(): iterable {
    yield new Missing();
}
`,
		"firstClassCallableGapPreserved": `<?php
function run(): \Closure {
    return (new class {
        public function m(): void {
            new Missing();
        }
    })->m(...);
}
`,
		"statementBodyDeclarationFound": `<?php
function run($flag): void {
    if ($flag) {
        class InlineDecl {
            public function m(): void {
                new Missing();
            }
        }
    }
}
`,
		"clean": `<?php
class Real {
    const FIELD = 1;
    public int $field = 1;
    public function bar(): void {}
    public static function staticBar(): void {}
}
function run(Real $r): void {
    new Real();
    Real::staticBar();
    Real::FIELD;
    $r->bar();
    echo $r->field;
}
`,
	}

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"unresolvedInstantiation": {
			{Message: "Instantiated class Missing not found.", Line: 3, Column: 5},
		},
		"abstractInstantiation": {
			{Message: "Instantiated class Base is abstract.", Line: 4, Column: 5},
		},
		"interfaceInstantiation": {
			{Message: "Cannot instantiate interface HasRun.", Line: 4, Column: 5},
		},
		"unresolvedFunctionCall": {
			{Message: "Function missingFunction not found.", Line: 2, Column: 1},
		},
		"classicStaticCall": {
			{Message: "Call to an undefined static method Real::missing().", Line: 7, Column: 5},
			{Message: "Call to static method bar() on an unknown class Missing.", Line: 8, Column: 5},
		},
		"dynamicStaticCall": {
			{Message: "Access to undefined static property Real::$.", Line: 7, Column: 5},
		},
		"instanceMethodCall": {},
		"thisMethodCall": {
			{Message: "Call to an undefined method Real::missing().", Line: 6, Column: 9},
			{Message: "Using $this inside static method Real::staticRun().", Line: 9, Column: 9},
		},
		"classConstFetch": {
			{Message: "Access to undefined constant Real::MISSING.", Line: 7, Column: 10},
			{Message: "Access to constant Missing::FIELD on an unknown class Missing.", Line: 8, Column: 10},
		},
		"propertyFetch":         {},
		"chainedMethodCalls":    {},
		"chainedPropertyAccess": {},
		"arrowFunctionKeepsEnclosingCurrentFn": {
			{Message: "Using $this inside static method Real::staticRun().", Line: 5, Column: 23},
		},
		"switchBodyGapPreserved":         {},
		"yieldExpressionGapPreserved":    {},
		"firstClassCallableGapPreserved": {},
		"statementBodyDeclarationFound":  {},
		"clean":                          {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, guards, _ := buildTypeRefTestContext(t, filename, src)
			got := sortIssuesForCompare(CheckSymbolIssuesFromCST(filename, []byte(src), ctx, guards))

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

func TestCheckSymbolIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckSymbolIssuesFromCST("empty.php", nil, &AnalysisContext{}, reflectionGuards{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}
