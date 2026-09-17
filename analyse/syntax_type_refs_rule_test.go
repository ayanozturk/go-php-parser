package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// buildTypeRefTestContext parses src, builds a single-file project-index
// resolver (mirroring runAnalysisLevelOnFiles's setup), and computes
// reflectionGuards once so both the ast.Node path and the CST-direct path
// can be compared against the identical guards value.
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

func TestCheckTypeReferenceIssuesFromCSTMatchesASTPath(t *testing.T) {
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

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, guards, nodes := buildTypeRefTestContext(t, filename, src)
			want := sortIssuesForCompare((&Level0Rule{}).checkTypeReferences(filename, nodes, ctx, fileCtx))
			got := sortIssuesForCompare(CheckTypeReferenceIssuesFromCST(filename, []byte(src), ctx, guards))
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

func TestCheckTypeReferenceIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckTypeReferenceIssuesFromCST("empty.php", nil, &AnalysisContext{}, reflectionGuards{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}

// TestCheckTypeReferenceIssuesFromCSTClassConstantsGapIsPreserved documents a
// pre-existing production gap: walkAllConfigured's *ast.ClassNode case
// (phpstan_level0_walk.go) only recurses into n.Properties/n.Methods, never
// n.Constants, so class-member constants are invisible to every rule driven
// by that dispatcher today — including checkTypeReferenceOnNode. The
// CST-direct port deliberately preserves this (skips KindClassConstDecl) to
// stay parity-safe; both paths must agree on zero issues here even though an
// unresolved type is present, until the dispatcher gap is fixed upstream.
func TestCheckTypeReferenceIssuesFromCSTClassConstantsGapIsPreserved(t *testing.T) {
	filename := "classConstGap.php"
	src := `<?php
class C {
    const MissingType FIELD = null;
}
`
	ctx, fileCtx, guards, nodes := buildTypeRefTestContext(t, filename, src)
	want := (&Level0Rule{}).checkTypeReferences(filename, nodes, ctx, fileCtx)
	got := CheckTypeReferenceIssuesFromCST(filename, []byte(src), ctx, guards)
	if len(want) != 0 || len(got) != 0 {
		t.Fatalf("expected both paths to report zero issues (preserved gap), ast=%d cst=%d", len(want), len(got))
	}
}
