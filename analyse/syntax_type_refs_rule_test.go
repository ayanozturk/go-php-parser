package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// buildTypeRefTestContext parses src, builds a single-file project-index
// resolver (mirroring runAnalysisLevelOnFiles's setup), and computes
// reflectionGuards once. fileCtx and nodes are still returned for other test
// files in this package that build their own ast.Node-side comparisons on
// top of this helper.
func buildTypeRefTestContext(t *testing.T, filename, src string) (ctx *AnalysisContext, fileCtx FileTypeContext, guards reflectionGuards, nodes []ast.Node) {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(src))
	if len(diags) > 0 {
		t.Fatalf("unexpected parse diagnostics for %s: %v", filename, diags)
	}
	fileCtx = CollectFileTypeContext(nodes)
	project := BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	ctx = &AnalysisContext{Resolver: project}
	guards = collectReflectionGuards(nodes, nil, fileCtx)
	return ctx, fileCtx, guards, nodes
}

func TestCheckTypeReferenceIssuesFromCST(t *testing.T) {
	cases := map[string]string{
		"unresolvedUseFunctionAndConst": `<?php
use function Missing\missing_fn;
use const Missing\MISSING_CONST;
`,
		"unresolvedParamAndReturnType": `<?php
function run(MissingParam $p): MissingReturn {}
`,
		"unresolvedClosureParamType": `<?php
function outer(): void {
    $fn = function (MissingClosureParam $p): MissingClosureReturn {
        return $p;
    };
}
`,
		"unresolvedInterfaceMethodTypes": `<?php
interface HasRun {
    public function run(MissingParam $p): MissingReturn;
}
`,
		"unresolvedPropertyType": `<?php
class C {
    public MissingType $field;
}
`,
		"unresolvedGlobalConstType": `<?php
const MissingType FIELD = null;
`,
		"caughtUnknownClass": `<?php
try {
    doSomething();
} catch (MissingException $e) {
}
`,
		"caughtNonThrowable": `<?php
trait NotThrowable {}
try {
    doSomething();
} catch (NotThrowable $e) {
}
`,
		"unresolvedAttributeClass": `<?php
#[MissingAttribute]
class C {}
`,
		"unresolvedAttributeOnMember": `<?php
class C {
    #[MissingAttribute]
    public int $field;
}
`,
		"classExistsGuardSuppresses": `<?php
function run(): void {
    if (class_exists(MissingButGuarded::class)) {
        return;
    }
}
class Uses {
    public MissingButGuarded $field;
}
`,
		"multiNamespaceDifferentAliases": `<?php
namespace App\First;

use App\First\RealDep as Shared;

class A {
    public Shared $field;
}

namespace App\Second;

use App\Second\OtherRealDep as Shared;

class B {
    public Shared $field;
}
`,
		"clean": `<?php
class Real {}
class C {
    public Real $field;
    public function run(Real $p): Real {
        return $p;
    }
}
`,
		"switchBodyGapPreserved": `<?php
function run($x): void {
    switch ($x) {
        case 1:
            try {
                doSomething();
            } catch (MissingInSwitch $e) {
            }
            break;
    }
}
`,
		"statementBodyDeclarationFound": `<?php
function run(bool $flag): void {
    if ($flag) {
        class InlineDecl {
            public function m() {
                try {
                    doSomething();
                } catch (MissingInInlineDecl $e) {
                }
            }
        }
    }
}
`,
		"anonymousClassInStatementBodyFound": `<?php
function run(): object {
    return new class {
        public function m(MissingAnonParam $p): void {
        }
    };
}
`,
		"yieldExpressionGapPreserved": `<?php
function run() {
    yield from (function (MissingYieldParam $p) {
        return $p;
    })();
}
`,
		"enumCaseAttributeGapPreserved": `<?php
enum SomeEnum {
    #[MissingEnumCaseAttribute]
    case Beta;
}
`,
		"attributeBeforeStatementBodyExpressionGapPreserved": `<?php
function run(): \Closure {
    return #[MissingFloatingAttribute] function () {
    };
}
`,
		"firstClassCallableGapPreserved": `<?php
function run(): \Closure {
    return (new class {
        public function m(): MissingFCCReturn {
        }
    })->m(...);
}
`,
		"closureParamAttributeFoundEvenNestedInCall": `<?php
function run(): void {
    doSomething(function (
        #[MissingNestedClosureParamAttribute] string $p,
    ): void {
    });
}
`,
		"classConstantAttributeGapPreserved": `<?php
class C {
    #[MissingClassConstAttribute]
    const FIELD = 1;
}
`,
	}

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"unresolvedUseFunctionAndConst": {
			{Message: `Used function Missing\missing_fn not found.`, Line: 2, Column: 1},
			{Message: `Used constant Missing\MISSING_CONST not found.`, Line: 3, Column: 1},
		},
		"unresolvedParamAndReturnType": {
			{Message: "Return type references unknown class MissingReturn.", Line: 2, Column: 1},
			{Message: "Parameter $p references unknown class MissingParam.", Line: 2, Column: 14},
		},
		"unresolvedClosureParamType": {
			{Message: "Return type references unknown class MissingClosureReturn.", Line: 3, Column: 11},
			{Message: "Parameter $p references unknown class MissingClosureParam.", Line: 3, Column: 21},
		},
		"unresolvedInterfaceMethodTypes": {
			{Message: "Return type references unknown class MissingReturn.", Line: 3, Column: 12},
			{Message: "Parameter $p references unknown class MissingParam.", Line: 3, Column: 25},
		},
		"unresolvedPropertyType": {
			{Message: "Property $field references unknown class MissingType.", Line: 3, Column: 5},
		},
		"unresolvedGlobalConstType": {},
		"caughtUnknownClass": {
			{Message: "Caught class MissingException not found.", Line: 4, Column: 3},
		},
		"caughtNonThrowable": {
			{Message: "Caught trait NotThrowable is not throwable.", Line: 5, Column: 3},
		},
		"unresolvedAttributeClass": {
			{Message: "Attribute class MissingAttribute not found.", Line: 2, Column: 3},
		},
		"unresolvedAttributeOnMember": {
			{Message: "Attribute class MissingAttribute not found.", Line: 3, Column: 7},
		},
		"classExistsGuardSuppresses": {},
		"multiNamespaceDifferentAliases": {
			{Message: `Property $field references unknown class App\First\RealDep.`, Line: 7, Column: 5},
			{Message: `Property $field references unknown class App\Second\OtherRealDep.`, Line: 15, Column: 5},
		},
		"clean":                         {},
		"switchBodyGapPreserved":        {},
		"statementBodyDeclarationFound": {},
		"anonymousClassInStatementBodyFound": {
			{Message: "Parameter $p references unknown class MissingAnonParam.", Line: 4, Column: 27},
		},
		"yieldExpressionGapPreserved":                        {},
		"enumCaseAttributeGapPreserved":                      {},
		"attributeBeforeStatementBodyExpressionGapPreserved": {},
		"firstClassCallableGapPreserved":                     {},
		"closureParamAttributeFoundEvenNestedInCall": {
			{Message: "Attribute class MissingNestedClosureParamAttribute not found.", Line: 4, Column: 11},
		},
		"classConstantAttributeGapPreserved": {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, guards, _ := buildTypeRefTestContext(t, filename, src)
			got := sortIssuesForCompare(CheckTypeReferenceIssuesFromCST(filename, []byte(src), ctx, guards))

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

func TestCheckTypeReferenceIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckTypeReferenceIssuesFromCST("empty.php", nil, &AnalysisContext{}, reflectionGuards{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

// TestCheckTypeReferenceIssuesFromCSTClassConstantsGapIsPreserved documents a
// pre-existing production gap: class-member constants are invisible to the
// type-reference check today, so an unresolved type used only in a class
// constant declaration is not reported.
func TestCheckTypeReferenceIssuesFromCSTClassConstantsGapIsPreserved(t *testing.T) {
	filename := "classConstGap.php"
	src := `<?php
class C {
    const MissingType FIELD = null;
}
`
	ctx, _, guards, _ := buildTypeRefTestContext(t, filename, src)
	got := CheckTypeReferenceIssuesFromCST(filename, []byte(src), ctx, guards)
	if len(got) != 0 {
		t.Fatalf("expected preserved gap to report zero issues, got %d: %+v", len(got), got)
	}
}
