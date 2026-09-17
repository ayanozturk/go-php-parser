package analyse

import (
	"os"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// analysisRuleRegistrySnapshot captures the production rule registry as
// populated by every rule file's init(). TestMain runs after all init()
// functions in the test binary have completed (both production rule files'
// and test files'), so this is guaranteed to see the full registry before
// any test has a chance to call ClearAnalysisRules().
//
// Some tests in rules_test.go (TestListRegisteredAnalysisRuleCodes,
// TestClearAnalysisRules, TestRunAnalysisRulesDeterministicOrder) clear the
// registry and/or register their own fake single-purpose rules without
// restoring the original registry afterward. Depending on test ordering
// (full run, or a narrower -run filter) this can leave the registry either
// empty or polluted with test-only fake rules for the remainder of a run.
// This snapshot lets TestRunAnalysisRulesWithContextCST restore the
// registry to the known-good production state itself, unconditionally,
// rather than only when it happens to be empty - so it exercises the real
// production pipeline regardless of what ran before it, instead of either
// silently degrading to "zero rules registered, zero issues found" or
// silently running against a polluted registry.
var analysisRuleRegistrySnapshot map[string]analysisRuleEntry

func TestMain(m *testing.M) {
	analysisRuleRegistryLock.RLock()
	analysisRuleRegistrySnapshot = make(map[string]analysisRuleEntry, len(analysisRuleRegistry))
	for k, v := range analysisRuleRegistry {
		analysisRuleRegistrySnapshot[k] = v
	}
	analysisRuleRegistryLock.RUnlock()

	os.Exit(m.Run())
}

// restoreAnalysisRuleRegistry unconditionally resets the global analysis
// rule registry to the TestMain-captured production snapshot, discarding
// anything another test left behind (missing entries from an unrestored
// ClearAnalysisRules call, or stray fake rules from an unrestored
// RegisterAnalysisRule call). It is intentionally not gated on "registry is
// empty" - a non-empty but polluted registry is just as unsafe for this
// test as an empty one, and gating on emptiness only worked by accident of
// file-name/test ordering.
func restoreAnalysisRuleRegistry(t *testing.T) {
	t.Helper()
	analysisRuleRegistryLock.Lock()
	defer analysisRuleRegistryLock.Unlock()
	for k := range analysisRuleRegistry {
		delete(analysisRuleRegistry, k)
	}
	for k, v := range analysisRuleRegistrySnapshot {
		analysisRuleRegistry[k] = v
	}
	sortedRuleCodesDirty = true
}

// TestRunAnalysisRulesWithContextCST drives the WHOLE registered-rule
// pipeline (RunAnalysisRulesWithContext, not a single rule in isolation)
// with ctx.Content set (the CST-direct fused walk added by
// ensureSharedFileDiagnosticsFromCST/ensureStructuralIssuesFromCST), across
// multiple analysis levels, and asserts the resulting issue set against
// golden expected values. Rules outside the fused walk (arg count/type,
// deprecated calls, level1 variables, level2 method existence/non-object,
// level6/7/8 method checks, property type, unreachable code, etc.) are
// exercised too since they're part of the same registry - this proves
// ctx.Content drives the full production pipeline correctly, not just that
// the 10 ported checks individually produce correct output in isolation.
//
// NOTE: this test depends on the global analysis-rule registry populated by
// each rule file's init(). Some other tests in this package call
// ClearAnalysisRules() without restoring the registry afterward
// (TestListRegisteredAnalysisRuleCodes, TestClearAnalysisRules in
// rules_test.go), which can make this test fail depending on run order
// within `go test ./...` even though it passes in isolation
// (`go test -run TestRunAnalysisRulesWithContextCST`). This is a
// pre-existing test-isolation hazard, not something introduced here.
func TestRunAnalysisRulesWithContextCST(t *testing.T) {
	restoreAnalysisRuleRegistry(t)

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

	type wantIssue struct {
		Code    string
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"mixedClassAndFunctionIssues/nilLevel": {
			{Code: "Level6.MissingPropertyType", Message: "Property $field has no type specified.", Line: 6, Column: 5},
			{Code: "Level6.MissingParameterType", Message: "Parameter $x has no type specified.", Line: 7, Column: 25},
			{Code: "A.RETURN.TYPE", Message: "Function f: return type mismatch, declared: int, actual: [string] at 14:1", Line: 14, Column: 1},
		},
		"mixedClassAndFunctionIssues/level0": {},
		"mixedClassAndFunctionIssues/level2": {},
		"mixedClassAndFunctionIssues/level6": {
			{Code: "Level6.MissingPropertyType", Message: "Property $field has no type specified.", Line: 6, Column: 5},
			{Code: "Level6.MissingParameterType", Message: "Parameter $x has no type specified.", Line: 7, Column: 25},
			{Code: "A.RETURN.TYPE", Message: "Function f: return type mismatch, declared: int, actual: [string] at 14:1", Line: 14, Column: 1},
		},
		"undefinedClassAndGoto/nilLevel": {
			{Code: "Level6.MissingReturnType", Message: "Function or method run has no return type specified.", Line: 2, Column: 1},
			{Code: "Level0.Symbols", Message: "Instantiated class MissingClass not found.", Line: 4, Column: 5},
		},
		"undefinedClassAndGoto/level0": {
			{Code: "Level0.Symbols", Message: "Instantiated class MissingClass not found.", Line: 4, Column: 5},
		},
		"undefinedClassAndGoto/level2": {
			{Code: "Level0.Symbols", Message: "Instantiated class MissingClass not found.", Line: 4, Column: 5},
		},
		"undefinedClassAndGoto/level6": {
			{Code: "Level6.MissingReturnType", Message: "Function or method run has no return type specified.", Line: 2, Column: 1},
			{Code: "Level0.Symbols", Message: "Instantiated class MissingClass not found.", Line: 4, Column: 5},
		},
		"phpDocAndMissingTypes/nilLevel": {
			{Code: "Level6.MissingPropertyType", Message: "Property $untyped has no type specified.", Line: 3, Column: 5},
			{Code: "Level6.MissingReturnType", Message: "Function or method run has no return type specified.", Line: 7, Column: 12},
		},
		"phpDocAndMissingTypes/level0": {},
		"phpDocAndMissingTypes/level2": {},
		"phpDocAndMissingTypes/level6": {
			{Code: "Level6.MissingPropertyType", Message: "Property $untyped has no type specified.", Line: 3, Column: 5},
			{Code: "Level6.MissingReturnType", Message: "Function or method run has no return type specified.", Line: 7, Column: 12},
		},
		"cleanFile/nilLevel": {},
		"cleanFile/level0":   {},
		"cleanFile/level2":   {},
		"cleanFile/level6":   {},
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

				cstCtx := &AnalysisContext{Resolver: project, AnalysisLevel: level, Content: []byte(src)}
				got := sortIssuesForCompare(RunAnalysisRulesWithContext(filename, nodes, cstCtx))

				wantIssues := want[name+"/"+levelName]
				if len(wantIssues) != len(got) {
					t.Fatalf("issue count mismatch: want=%d got=%d\nwant=%+v\ngot=%+v", len(wantIssues), len(got), wantIssues, got)
				}
				for i := range wantIssues {
					if wantIssues[i].Code != got[i].Code || wantIssues[i].Line != got[i].Line || wantIssues[i].Column != got[i].Column || wantIssues[i].Message != got[i].Message {
						t.Fatalf("issue %d mismatch:\nwant=%+v\ngot=%+v", i, wantIssues[i], got[i])
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
