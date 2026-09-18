# CST-Direct Shared-Parse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Parse each file's CST once per diagnostics pass instead of up to ~10 times, by threading one shared `*syntax.ParseResult` through the `Check*IssuesFromCST` functions the two fused funnels call, instead of each function calling `syntax.Parse(content)` itself.

**Architecture:** No new architecture — this is a pure internal refactor with no observable behavior change. `syntax.Parse` is deterministic (same bytes in, same tree out), so calling it once and sharing the result instead of calling it N times must produce identical diagnostics; the whole plan's validation strategy leans on that fact (corpus-diff must show 0 mismatches, not "acceptable new diagnostics" — any diff here means a wiring bug, not a real fix, unlike some tasks in the prior Phase 5 cleanup plan). Each of the 10 affected `Check*IssuesFromCST` functions gets split into a thin public wrapper (unchanged signature, still calls `syntax.Parse` — kept for external/differential-test callers that only have raw bytes) and a new unexported `...FromParsed` function that takes the already-parsed `*syntax.ParseResult` directly. The two fused funnels (`ensureSharedFileDiagnosticsFromCST`, `ensureStructuralIssuesFromCST` in `analyse/syntax_fused_rule.go`) parse once and call the `...FromParsed` variants for all their sub-checks.

**Tech Stack:** Go 1.23, this repo's `analyse`/`syntax` packages, `cmd/analysis-corpus-snapshot` + `test_projects/{composer-src,symfony}` for corpus-diff correctness validation, `testing.B` Go benchmarks for the performance claim.

**Spec:** This plan is its own spec, produced by a live audit of `analyse/syntax_fused_rule.go` and the 10 `Check*IssuesFromCST` functions it calls (see `grep -n "syntax.Parse(" analyse/*.go` — 10 non-test call sites outside the 3 already-single-parse pilot rules `AssignmentInConditionRule`/`SideEffectsRule`/nothing-else, which are out of scope here since they each already parse exactly once per their own registered-rule invocation, not inside a funnel that calls them repeatedly). Background/motivation: this reparse cost was a known, explicitly-deferred concern from the prior "CST-direct migration Phase 4/5" work (see `AGENTS.md`'s "CST-direct migration" section) — it was fine while `ctx.Content` was opt-in, but Phase 5 made the CST-direct path unconditional in production, so this cost is now permanent, unconditional, and worth fixing.

## Global Constraints

- **Zero corpus-diff mismatches, no exceptions.** Unlike some tasks in the Phase 5 cleanup plan this follows, there is no "additive true-positive fix" excuse available here — this is a pure performance refactor. Any diagnostic difference on `test_projects/composer-src`/`test_projects/symfony` before vs. after means a real bug was introduced, not a discovery to celebrate. Use `cmd/analysis-corpus-snapshot` exactly as in the prior plan (absolute paths, `test_projects` is gitignored: `/Users/ayan.ozturk/rg/go-php-parser/test_projects/composer-src`, `/Users/ayan.ozturk/rg/go-php-parser/test_projects/symfony`).
- **Every public `Check*IssuesFromCST` function keeps its exact existing exported signature.** Differential tests (rewritten in the prior Phase 5 plan's Task 5 to assert golden CST-only output) call these by name with `(filename, content []byte, ...)` — do not change their signatures, only their bodies (delegate to the new unexported `...FromParsed` sibling).
- **The new `...FromParsed` functions must each independently guard against `res == nil || res.File == nil || res.File.Root == nil`** (don't assume the funnel's one nil-check up front is the only place this can be reached from — some of the pilot/standalone rules or future callers might call an `...FromParsed` function directly).
- Run `go build ./...`, `go vet ./...`, and `go test ./...` after every task — expect exactly 2 pre-existing unrelated failures (`TestPropertyHooksDeclBindsNameOnly`, `TestDynamicBracedStaticMemberWalksNestedPropertyRef` in `analyse/syntax_binder_test.go`) and no new failures.
- Follow existing code style: no narrative comments explaining what a line does; doc-comments on new exported-adjacent functions are fine and expected (that's how the existing `Check*IssuesFromCST` functions are documented).

---

### Task 1: Split the 10 `Check*IssuesFromCST` functions and thread one shared parse through both funnels

**Files:**
- Modify: `analyse/syntax_class_model_rule.go` (`CheckClassModelIssuesFromCST`)
- Modify: `analyse/syntax_language_rule.go` (`CheckLanguageIssuesFromCST`)
- Modify: `analyse/syntax_method_visibility_rule.go` (`CheckMethodVisibilityIssuesFromCST`)
- Modify: `analyse/syntax_missing_types_rule.go` (`CheckMissingTypeIssuesFromCST`)
- Modify: `analyse/syntax_phpdoc_rule.go` (`CheckPHPDocIssuesFromCST`)
- Modify: `analyse/syntax_property_callable_rule.go` (`CheckPropertyCallableTypeIssuesFromCST`)
- Modify: `analyse/syntax_return_type_rule.go` (`CheckReturnTypeIssuesFromCST`)
- Modify: `analyse/syntax_symbols_rule.go` (`CheckSymbolIssuesFromCST`)
- Modify: `analyse/syntax_throw_type_rule.go` (`CheckThrowTypeIssuesFromCST`)
- Modify: `analyse/syntax_type_refs_rule.go` (`CheckTypeReferenceIssuesFromCST`)
- Modify: `analyse/syntax_fused_rule.go` (`checkEmptyStatementIssuesFromCST`, `ensureSharedFileDiagnosticsFromCST`, `ensureStructuralIssuesFromCST`)

**Interfaces:**
- Consumes: `syntax.Parse(content []byte) *syntax.ParseResult` (existing, `syntax/parser.go:51`), `syntax.ParseResult{File *syntax.File, Diagnostics []syntax.Diagnostic, PHPVersion string}` (existing, `syntax/api.go:17`).
- Produces: for each of the 10 functions, a new unexported sibling named `check<Name>FromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext, ...same extra params as the original...) []AnalysisIssue` — e.g. `CheckClassModelIssuesFromCST(filename string, content []byte, ctx *AnalysisContext)` gets a sibling `checkClassModelIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext)`. Later tasks/callers should prefer the `...FromParsed` sibling when they already have a `*syntax.ParseResult` in hand.

This is 10 structurally-identical splits. Do them as one batch (same shape, same risk profile), not 10 separate reviews — but commit in 2 groups so a build break is easy to bisect: group A (the first 5 alphabetically: class_model, language, method_visibility, missing_types, phpdoc), group B (the remaining 5: property_callable, return_type, symbols, throw_type, type_refs), then the funnel wiring as a third commit.

- [ ] **Step 1: Read one function fully to confirm the exact split pattern before touching any of them**

```bash
sed -n '1,50p' analyse/syntax_class_model_rule.go
```

Every one of the 10 functions starts with the same shape:
```go
func CheckClassModelIssuesFromCST(filename string, content []byte, ctx *AnalysisContext) []AnalysisIssue {
	res := syntax.Parse(content)
	if res.File == nil || res.File.Root == nil {
		return nil
	}
	// ... the rest of the function body uses res.File.Root, ctx, and other params ...
}
```

The split is: keep the public function's signature and doc comment exactly as-is, but replace its body with a call to the new sibling:

```go
func CheckClassModelIssuesFromCST(filename string, content []byte, ctx *AnalysisContext) []AnalysisIssue {
	return checkClassModelIssuesFromParsed(filename, syntax.Parse(content), ctx)
}

func checkClassModelIssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	// ... the exact original body, unchanged, starting from what used to be
	// right after the nil-check ...
}
```

- [ ] **Step 2: Apply this exact split to group A (5 files: class_model, language, method_visibility, missing_types, phpdoc)**

For each file, the only change is: (a) the public function's body becomes the two-line delegation shown above, (b) a new function is added directly below it with the `FromParsed` name and the original body (nil-check included, defensively, even though the funnel will also nil-check once up front — per Global Constraints, each `...FromParsed` function must guard itself). Preserve every parameter beyond `content`/`res` exactly (e.g. `CheckSymbolIssuesFromCST` also takes `guards reflectionGuards`; `CheckPHPDocIssuesFromCST` also takes `phpDocAliases map[string]struct{}` — carry these through to the `...FromParsed` sibling unchanged).

- [ ] **Step 3: Build and run the package tests for group A**

```bash
go build ./analyse/... && go test ./analyse/... 2>&1 | tail -30
```

Expect only the 2 pre-existing unrelated failures.

- [ ] **Step 4: Commit group A**

```bash
git add analyse/syntax_class_model_rule.go analyse/syntax_language_rule.go analyse/syntax_method_visibility_rule.go analyse/syntax_missing_types_rule.go analyse/syntax_phpdoc_rule.go
git commit -m "analyse: split 5 Check*IssuesFromCST functions into parse + FromParsed (group A)"
```

- [ ] **Step 5: Apply the same split to group B (5 files: property_callable, return_type, symbols, throw_type, type_refs)**, then build/test/commit exactly as steps 2-4.

```bash
go build ./analyse/... && go test ./analyse/... 2>&1 | tail -30
git add analyse/syntax_property_callable_rule.go analyse/syntax_return_type_rule.go analyse/syntax_symbols_rule.go analyse/syntax_throw_type_rule.go analyse/syntax_type_refs_rule.go
git commit -m "analyse: split 5 Check*IssuesFromCST functions into parse + FromParsed (group B)"
```

- [ ] **Step 6: Split `checkEmptyStatementIssuesFromCST` the same way** (`analyse/syntax_fused_rule.go`, currently lines ~11-24 — re-read the file fresh, line numbers may have drifted since the audit):

```go
func checkEmptyStatementIssuesFromCST(filename string, content []byte) []AnalysisIssue {
	return checkEmptyStatementIssuesFromParsed(filename, syntax.Parse(content))
}

func checkEmptyStatementIssuesFromParsed(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		if n.Kind() == syntax.KindEmptyStmt {
			issues = append(issues, issueSpanRed(filename, n, emptyStatementCode, "Empty statement detected"))
		}
		return true
	})
	return issues
}
```

- [ ] **Step 7: Rewrite `ensureSharedFileDiagnosticsFromCST` to parse once and call the `...FromParsed` siblings**

Current shape (re-read `analyse/syntax_fused_rule.go` fresh before editing — this plan quotes it as of the prior Phase 5 cleanup's final state, but re-verify):

```go
func ensureSharedFileDiagnosticsFromCST(filename string, content []byte, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	fileCtx := analysisFileTypeContext(ctx, nodes)
	guards := collectReflectionGuards(nodes, ctx, fileCtx)

	collectStructural := analysisLevelAtLeast(ctx, 2)
	collectReturn := collectStructural
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)
	if collectStructural && ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}

	var issues []AnalysisIssue
	issues = append(issues, CheckClassModelIssuesFromCST(filename, content, ctx)...)
	issues = append(issues, CheckTypeReferenceIssuesFromCST(filename, content, ctx, guards)...)
	issues = append(issues, CheckSymbolIssuesFromCST(filename, content, ctx, guards)...)
	issues = append(issues, CheckLanguageIssuesFromCST(filename, content)...)
	ctx.level0PropertyCallableIssues = CheckPropertyCallableTypeIssuesFromCST(filename, content)
	ctx.emptyStatementIssues = checkEmptyStatementIssuesFromCST(filename, content)

	if collectStructural {
		ctx.methodVisibilityIssues = CheckMethodVisibilityIssuesFromCST(filename, content, ctx)
		ctx.throwTypeIssues = CheckThrowTypeIssuesFromCST(filename, content, ctx)
		ctx.phpDocIssues = CheckPHPDocIssuesFromCST(filename, content, ctx, ctx.phpDocTypeAliases)
		if collectMissingTypes {
			ctx.missingTypeIssues = CheckMissingTypeIssuesFromCST(filename, content, ctx)
		}
		if collectReturn {
			ctx.returnTypeIssues = CheckReturnTypeIssuesFromCST(filename, content, ctx)
		}
	}

	ctx.level0Issues = issues
	ctx.hasLevel0Issues = true
	if collectStructural {
		ctx.hasStructuralIssues = true
		if collectReturn {
			ctx.hasReturnTypeIssues = true
		}
	}
	return ctx
}
```

New shape — add one `res := syntax.Parse(content)` at the top and swap every `Check*IssuesFromCST(filename, content, ...)` call for `check*IssuesFromParsed(filename, res, ...)` (note the lowercase-first-letter unexported name), and `checkEmptyStatementIssuesFromCST(filename, content)` for `checkEmptyStatementIssuesFromParsed(filename, res)`:

```go
func ensureSharedFileDiagnosticsFromCST(filename string, content []byte, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	res := syntax.Parse(content)
	fileCtx := analysisFileTypeContext(ctx, nodes)
	guards := collectReflectionGuards(nodes, ctx, fileCtx)

	collectStructural := analysisLevelAtLeast(ctx, 2)
	collectReturn := collectStructural
	collectMissingTypes := analysisLevelAtLeast(ctx, 6)
	if collectStructural && ctx.phpDocTypeAliases == nil {
		ctx.phpDocTypeAliases = collectPHPDocTypeAliases(nodes)
	}

	var issues []AnalysisIssue
	issues = append(issues, checkClassModelIssuesFromParsed(filename, res, ctx)...)
	issues = append(issues, checkTypeReferenceIssuesFromParsed(filename, res, ctx, guards)...)
	issues = append(issues, checkSymbolIssuesFromParsed(filename, res, ctx, guards)...)
	issues = append(issues, checkLanguageIssuesFromParsed(filename, res)...)
	ctx.level0PropertyCallableIssues = checkPropertyCallableTypeIssuesFromParsed(filename, res)
	ctx.emptyStatementIssues = checkEmptyStatementIssuesFromParsed(filename, res)

	if collectStructural {
		ctx.methodVisibilityIssues = checkMethodVisibilityIssuesFromParsed(filename, res, ctx)
		ctx.throwTypeIssues = checkThrowTypeIssuesFromParsed(filename, res, ctx)
		ctx.phpDocIssues = checkPHPDocIssuesFromParsed(filename, res, ctx, ctx.phpDocTypeAliases)
		if collectMissingTypes {
			ctx.missingTypeIssues = checkMissingTypeIssuesFromParsed(filename, res, ctx)
		}
		if collectReturn {
			ctx.returnTypeIssues = checkReturnTypeIssuesFromParsed(filename, res, ctx)
		}
	}

	ctx.level0Issues = issues
	ctx.hasLevel0Issues = true
	if collectStructural {
		ctx.hasStructuralIssues = true
		if collectReturn {
			ctx.hasReturnTypeIssues = true
		}
	}
	return ctx
}
```

**Verify the exact unexported function name each split step actually produced** before writing these call sites — the naming convention above (`check<Name>IssuesFromParsed`, first letter lowercased, "Check" → "check") must match verbatim what Steps 2/5 named each sibling, or this won't compile. If you named any sibling differently in an earlier step, either rename it now to match this convention or adjust these call sites to match whatever you actually named it — pick one and be consistent across all 10.

- [ ] **Step 8: Rewrite `ensureStructuralIssuesFromCST` the same way** — parse once at the top, delegate to the same `...FromParsed` siblings for its 5 sub-checks (`methodVisibilityIssues`, `throwTypeIssues`, `phpDocIssues`, `missingTypeIssues`, `returnTypeIssues`).

- [ ] **Step 9: Build, vet, test**

```bash
go build ./... && go vet ./... && go test ./... 2>&1 | tail -40
```

Expect only the 2 pre-existing unrelated failures, nothing new.

- [ ] **Step 10: Corpus-diff validation — must be exactly 0 mismatches (no exceptions, per Global Constraints)**

```bash
go build -o /tmp/analysis-corpus-snapshot ./cmd/analysis-corpus-snapshot
/tmp/analysis-corpus-snapshot --root /Users/ayan.ozturk/rg/go-php-parser/test_projects/composer-src --output /tmp/shared-parse-baseline-composer.json
/tmp/analysis-corpus-snapshot --root /Users/ayan.ozturk/rg/go-php-parser/test_projects/symfony --output /tmp/shared-parse-baseline-symfony.json
```

Then check out the commit just before this task started (`git stash` your work first if needed, or just diff against `HEAD~3` — whichever 3 commits from Steps 4/5/8 you just made — to get a true before/after; the safest approach is to capture the baseline BEFORE Step 2 even starts, so re-order: actually run this baseline capture as your very first action, before Step 1, and only diff at the end). If you didn't capture a true pre-task baseline, capture one now from `git stash` on a clean pre-task checkout, then compare:

```bash
/tmp/analysis-corpus-snapshot --root /Users/ayan.ozturk/rg/go-php-parser/test_projects/composer-src --baseline /tmp/shared-parse-baseline-composer.json
/tmp/analysis-corpus-snapshot --root /Users/ayan.ozturk/rg/go-php-parser/test_projects/symfony --baseline /tmp/shared-parse-baseline-symfony.json
```

Expected: 0 mismatches on both. Any mismatch means a wiring bug (e.g. a call site left pointing at the old `Check*IssuesFromCST(filename, content, ...)` form instead of the new `check*IssuesFromParsed(filename, res, ...)` form, silently double-parsing one check while skipping the shared result for another) — find and fix it before proceeding, do not rationalize a diff here.

- [ ] **Step 11: Commit the funnel wiring**

```bash
git add analyse/syntax_fused_rule.go
git commit -m "analyse: parse once per file and share across all Check*FromCST calls in both fused funnels"
```

---

### Task 2: Prove the performance win with a real benchmark

**Files:**
- Create: `analyse/syntax_fused_rule_bench_test.go`

**Interfaces:**
- Consumes: `ensureSharedFileDiagnosticsFromCST` (existing, now-modified in Task 1), `syntax.Parse` (existing).
- Produces: a `go test -bench` benchmark comparing the before/after parse count, giving a real number to cite instead of "up to ~10x" as a guess.

- [ ] **Step 1: Write a benchmark that measures `ensureSharedFileDiagnosticsFromCST` end-to-end on a realistic fixture**

```go
package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func BenchmarkEnsureSharedFileDiagnosticsFromCST(b *testing.B) {
	content := []byte(benchmarkFixtureSource)
	nodes, _ := syntax.ParseAST(content)
	level := 6
	for i := 0; i < b.N; i++ {
		ctx := &AnalysisContext{Content: content, AnalysisLevel: &level}
		ensureSharedFileDiagnosticsFromCST("bench.php", content, nodes, ctx)
	}
}

const benchmarkFixtureSource = `<?php
// a moderately complex fixture exercising class model, type refs, symbols,
// language checks, property-callable, empty-statement, method-visibility,
// throw-type, phpdoc, missing-types, and return-type checks all in one file
namespace App;

interface Greetable {
    public function greet(): string;
}

abstract class Base implements Greetable {
    /** @var string */
    protected $name;

    public function __construct(string $name) {
        $this->name = $name;
    }

    abstract public function greet(): string;

    protected function throwsSomething(): void {
        throw new \RuntimeException('nope');
    }
}

class Person extends Base {
    public function greet(): string {
        return "Hello, {$this->name}!";
    }

    public function process(array $items): array {
        $result = [];
        foreach ($items as $item) {
            if ($item) {
                $result[] = strtoupper($item);
            }
        }
        return $result;
    }
}
`
```

Adjust the fixture if it doesn't actually exercise most of the 10 check families — the goal is a realistic per-file cost, not a trivial one-liner that makes the parse cost look artificially small relative to fixed overhead. Reuse an existing fixture from `analyse/syntax_fused_rule_test.go`'s 4 fixtures if one is already known to exercise a good spread (check that file first — it may already have exactly this kind of fixture you can import/reuse instead of writing a new one).

- [ ] **Step 2: Run the benchmark against the current (post-Task-1) code**

```bash
go test ./analyse/ -bench BenchmarkEnsureSharedFileDiagnosticsFromCST -benchtime=3x -run '^$' -v
```

Record the `ns/op` and `allocs/op` numbers.

- [ ] **Step 3: Temporarily revert to the pre-Task-1 shape to get a true before number**

```bash
git stash
git log --oneline -5  # find the commit right before Task 1's first commit
git checkout <pre-task-1-commit> -- analyse/syntax_fused_rule.go analyse/syntax_class_model_rule.go analyse/syntax_language_rule.go analyse/syntax_method_visibility_rule.go analyse/syntax_missing_types_rule.go analyse/syntax_phpdoc_rule.go analyse/syntax_property_callable_rule.go analyse/syntax_return_type_rule.go analyse/syntax_symbols_rule.go analyse/syntax_throw_type_rule.go analyse/syntax_type_refs_rule.go
go test ./analyse/ -bench BenchmarkEnsureSharedFileDiagnosticsFromCST -benchtime=3x -run '^$' -v
git checkout HEAD -- analyse/  # restore Task 1's changes
git stash pop  # restore your new benchmark file if it got stashed
```

Record the "before" `ns/op`/`allocs/op` numbers. Compute and note the real speedup ratio (expect something in the 3-8x range on `ns/op` given ~10 parses collapse to 1, though allocation/GC overhead don't scale perfectly linearly with parse count — report the real number, not a guess).

- [ ] **Step 4: Run full build/test one more time to confirm the working tree is back to Task 1's committed state plus the new benchmark file**

```bash
go build ./... && go test ./... 2>&1 | tail -20
git status --short  # should show only the new bench test file as untracked/added
```

- [ ] **Step 5: Commit the benchmark**

```bash
git add analyse/syntax_fused_rule_bench_test.go
git commit -m "analyse: add benchmark proving the shared-parse win"
```

- [ ] **Step 6: Update `AGENTS.md`'s note about the reparse cost** (search for "reparse cost" — added during the prior Phase 5 cleanup plan) to record the fix and the measured numbers from Step 3, replacing the "worth remeasuring as a separate follow-up" framing with the actual before/after result.

```bash
git add AGENTS.md
git commit -m "AGENTS.md: record the shared-parse fix and measured speedup"
```

---

## Self-Review Notes

- **Spec coverage:** Task 1 covers the actual refactor (all 10 functions + both funnels + the empty-statement helper); Task 2 covers proving it worked, both correctness (already covered by Task 1's corpus-diff) and performance (the actual point of doing this).
- **Placeholder scan:** Task 1's code snippets are complete and copy-pasteable modulo the explicit "re-read the file fresh, line numbers may have drifted" caveats, which are honest signals about audit staleness, not missing content. Task 2's benchmark fixture is a real, complete PHP snippet, not a stub.
- **Type consistency:** the `check<Name>IssuesFromParsed(filename string, res *syntax.ParseResult, ctx *AnalysisContext, ...)` naming and signature convention is used consistently across every call site in Task 1 Step 7/8's rewritten funnel bodies.
- **Known risk:** Task 1 Step 7 flags its own naming-consistency risk explicitly (the executor must verify Steps 2/5's actual chosen names match the call sites) rather than silently assuming it — this is the one place a real mismatch (typo, inconsistent casing) would surface as a compile error, which is the safest possible failure mode for this kind of rename.
