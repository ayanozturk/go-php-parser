package analyse

import (
	"testing"
)

func TestCheckSymbolIssuesFromCSTMatchesASTPath(t *testing.T) {
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

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, guards, nodes := buildTypeRefTestContext(t, filename, src)
			want := sortIssuesForCompare((&Level0Rule{}).checkSymbolsAndCalls(filename, nodes, ctx, fileCtx))
			got := sortIssuesForCompare(CheckSymbolIssuesFromCST(filename, []byte(src), ctx, guards))
			if len(want) != len(got) {
				t.Fatalf("issue count mismatch: ast=%d cst=%d\nast=%+v\ncst=%+v", len(want), len(got), want, got)
			}
			for i := range want {
				if want[i].Line != got[i].Line || want[i].Column != got[i].Column || want[i].Message != got[i].Message {
					t.Fatalf("issue %d mismatch:\nast=%+v\ncst=%+v", i, want[i], got[i])
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
