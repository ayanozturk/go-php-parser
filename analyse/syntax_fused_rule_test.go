package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// TestRunAnalysisRulesWithContextCSTMatchesASTPath is the Phase 4 parity
// guard: it drives the WHOLE registered-rule pipeline (RunAnalysisRulesWithContext,
// not a single rule in isolation) once with ctx.Content unset (ast.Node fused
// walk, the pre-Phase-4 behavior) and once with ctx.Content set (the new
// CST-direct fused walk added by ensureSharedFileDiagnosticsFromCST/
// ensureStructuralIssuesFromCST), across multiple analysis levels, and
// asserts identical issue sets. Rules outside the fused walk (arg count/
// type, deprecated calls, level1 variables, level2 method existence/non-
// object, level6/7/8 method checks, property type, unreachable code, etc.)
// are exercised too since they're part of the same registry - this proves
// ctx.Content is a safe, invisible-to-callers opt-in, not just that the 10
// ported checks individually match in isolation.
func TestRunAnalysisRulesWithContextCSTMatchesASTPath(t *testing.T) {
	cases := map[string]string{
		"mixedClassAndFunctionIssues": `<?php
class Base {
    protected static function helper(): void {}
}
class Caller extends Base {
    public $field;
    public function run($x): int {
        if ($x) {
            Base::helper();
        }
        return $x;
    }
}
function f(): int {
    return "x";
}
`,
		"undefinedClassAndGoto": `<?php
function run() {
    goto end;
    new MissingClass();
    end:
    echo 1;
}
`,
		"phpDocAndMissingTypes": `<?php
class C {
    public $untyped;
    /**
     * @param int $x
     */
    public function run($x) {
        return $x;
    }
}
`,
		"cleanFile": `<?php
class C {
    public int $field;
    public function run(int $x): int {
        return $x;
    }
}
`,
	}

	for name, src := range cases {
		for _, level := range []*int{nil, intLevelPtr(0), intLevelPtr(2), intLevelPtr(6)} {
			levelName := "nilLevel"
			if level != nil {
				levelName = levelSuffix(*level)
			}
			t.Run(name+"/"+levelName, func(t *testing.T) {
				filename := name + ".php"
				nodes, diags := syntax.ParseAST([]byte(src))
				if len(diags) > 0 {
					t.Fatalf("unexpected parse diagnostics: %v", diags)
				}
				project := BuildProjectIndex(map[string][]ast.Node{filename: nodes})

				astCtx := &AnalysisContext{Resolver: project, AnalysisLevel: level}
				want := sortIssuesForCompare(RunAnalysisRulesWithContext(filename, nodes, astCtx))

				cstCtx := &AnalysisContext{Resolver: project, AnalysisLevel: level, Content: []byte(src)}
				got := sortIssuesForCompare(RunAnalysisRulesWithContext(filename, nodes, cstCtx))

				if len(want) != len(got) {
					t.Fatalf("issue count mismatch: ast=%d cst=%d\nast=%+v\ncst=%+v", len(want), len(got), want, got)
				}
				for i := range want {
					if want[i].Code != got[i].Code || want[i].Line != got[i].Line || want[i].Column != got[i].Column || want[i].Message != got[i].Message {
						t.Fatalf("issue %d mismatch:\nast=%+v\ncst=%+v", i, want[i], got[i])
					}
				}
			})
		}
	}
}

func intLevelPtr(v int) *int { return &v }

func levelSuffix(v int) string {
	switch v {
	case 0:
		return "level0"
	case 2:
		return "level2"
	case 6:
		return "level6"
	default:
		return "levelOther"
	}
}
