# CST-Direct Migration Phase 5 Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the CST-direct analysis path (`ctx.Content` branch) the *only* path exercised in production and tests across every caller in `go-php-parser` and `vscode-php-strom`, then delete the now-dead `ast.Node`-walk implementations of the 13 already-CST-ported rules.

**Architecture:** No new architecture — this closes the gap between what `AGENTS.md`'s "CST-direct migration" section claimed ("Phase 4 done") and what a repo-wide audit found: only `vscode-php-strom/server/providers/diagnostics.go` actually sets `ctx.Content`. Every `go-php-parser` CLI/library caller and almost every test still takes the old `[]ast.Node` branch, so the old code cannot be deleted yet. This plan (1) flips every remaining caller onto `ctx.Content`, (2) closes two gaps the earlier phases missed (`AssignmentInConditionRule`/`SideEffectsRule` never wired their CST version into their registered callback at all; `appendReturnTypeOnNode` has a third call site that doesn't branch on `Content`), (3) migrates the handful of tests that call old functions directly, then (4) deletes the old functions.

**Tech Stack:** Go 1.23, this repo's existing `analyse`/`syntax` packages, `cmd/analysis-corpus-snapshot` and `test_projects/{composer-src,symfony}` for corpus-diff validation, `vscode-php-strom`'s `make test-server-dev`/`make test-server` for cross-repo validation.

**Spec:** This plan **is** the spec — it was produced by a live repo audit (see conversation history / `AGENTS.md` "CST-direct migration" section, `/Users/ayan.ozturk/rg/go-php-parser/AGENTS.md:26-344`) rather than a separate design doc. Read that section before starting Task 1; it explains why the CST/ast split exists at all.

## Global Constraints

- Every task that changes `RunAnalysisRulesWithContext` output must be validated with **zero diagnostic mismatches** on `test_projects/composer-src` and `test_projects/symfony` before being considered done (use `cmd/analysis-corpus-snapshot`, see Task 1 Step 2 for the exact invocation).
- Do not touch `vscode-php-strom/server/go.mod`'s pin during Tasks 1-6 (they're all `go-php-parser`-internal). Only Task 8 (final AGENTS.md update) may reference the sibling repo, and it makes no code changes there — the pin was already bumped to include Phase 4 in a prior session.
- Do not delete any old `ast.Node` function until every caller of `RunAnalysisRulesWithContext` sets `ctx.Content` (Tasks 1-3) AND every direct test caller of that specific old function has been migrated (Tasks 4-5). Deleting early breaks the build for callers not yet flipped.
- Every deleted test must have a replacement assertion — do not silently drop coverage for a diagnostic family.
- Follow existing code style: this codebase has no comments beyond doc-comments on exported funcs; don't add narrative comments explaining what a line does.
- Run `go build ./...` and `go test ./...` after every task, not just at the end.

---

### Task 1: Wire `ctx.Content` into every go-php-parser CLI/library caller

**Files:**
- Modify: `command/analyze.go:168-181` (first `RunAnalysisRulesWithContext` call site, inside the cached-index worker loop)
- Modify: `command/analyze.go:301-314` (second call site, inside the fresh-index worker loop)
- Modify: `command/file_processor.go:409-418` (`runAnalysis` helper)
- Modify: `cmd/benchmark/main.go:990-1022` (`runAnalysis`, `job` struct needs a `content []byte` field)
- Modify: `cmd/diagnostic-diff/main.go:211-228` (`runEngine` — already reads `content` via `os.ReadFile` at line 212, just needs to thread it into `ctx.Content`)
- Modify: `cmd/analysis-corpus-snapshot/main.go:150-193` (needs a `contents map[string][]byte` alongside the existing `parsed map[string][]ast.Node`, populated wherever files are currently read+parsed — read the ~40 lines above line 165 to find the parse loop before editing)
- Modify: `analyse/rules.go:130-137` (`RunAnalysisRules` wrapper passes `nil` ctx — leave as `nil`, since callers with no `AnalysisContext` at all have no content to give it either; this file needs no change, listed here only so the task doesn't miss checking it)

**Interfaces:**
- Consumes: `AnalysisContext.Content []byte` (already defined, `analyse/context.go:240`), `ensureSharedFileDiagnosticsFromCST`/`ensureStructuralIssuesFromCST` (already defined, `analyse/syntax_fused_rule.go`) — no new interfaces, this task only adds `ctx.Content = <bytes>` assignments before existing `RunAnalysisRulesWithContext` calls.
- Produces: every production/CLI caller now takes the CST-direct branch. Task 4/5/6 depend on this being complete for all of them.

- [ ] **Step 1: `command/analyze.go` — wire both call sites**

At line ~178 (inside the first worker loop, right after `ctx := snapshot.NewAnalysisContext()` and `ctx.AnalysisLevel = level`), add:

```go
ctx.Content = contents[path]
```

`contents` is the `map[string][]byte` already populated at `command/analyze.go:162` (`contents[file.path] = file.content`) and captured by the closure. Do the same at the second call site (~line 311); confirm a `contents` map exists in that function's scope — if not (the second call site's function may only build `parsed`, not `contents`), add one populated the same way the first function does, from whatever raw-bytes source that function's file-reading loop already produces (it parses files to build `parsed`, so raw bytes exist somewhere upstream — trace `parsed[file.path] = file.nodes` in that function backward to find the corresponding content variable).

- [ ] **Step 2: Run corpus-diff baseline before touching anything else**

```bash
cd /Users/ayan.ozturk/rg/go-php-parser
go build -o /tmp/analysis-corpus-snapshot ./cmd/analysis-corpus-snapshot
/tmp/analysis-corpus-snapshot --root test_projects/composer-src --output /tmp/baseline-composer.json
/tmp/analysis-corpus-snapshot --root test_projects/symfony --output /tmp/baseline-symfony.json
```

Keep these two JSON files — every subsequent step in this plan diffs against them with `--baseline`.

- [ ] **Step 3: `command/file_processor.go` — wire `runAnalysis`**

`runAnalysis(path string, nodes []ast.Node, project *analyse.ProjectIndex)` at line 409 has no raw bytes parameter today. Check its callers (`grep -n "runAnalysis(" command/*.go`) — they almost certainly already have the file's raw content in scope (it was read to produce `nodes`). Add a `content []byte` parameter to `runAnalysis`, thread it from every call site, and set:

```go
ctx := &analyse.AnalysisContext{Resolver: project, AnalysisLevel: configuredAnalysisLevel, Content: content}
```

- [ ] **Step 4: `cmd/benchmark/main.go` — add content to the job struct**

The `job` struct at line ~1004 is `struct { path string; nodes []ast.Node }`. Add `content []byte`, populate it wherever jobs are enqueued (find the producer side of `jobCh`), and set `ctx.Content = j.content` before the `RunAnalysisRulesWithContext` call at line 1019.

- [ ] **Step 5: `cmd/diagnostic-diff/main.go` — wire `runEngine`**

Line 212 already does `content, err := os.ReadFile(path)`. Add `ctx.Content = content` right after `ctx.AnalysisLevel = &level` (line ~225), before the `RunAnalysisRulesWithContext` call at line 227.

- [ ] **Step 6: `cmd/analysis-corpus-snapshot/main.go` — add a contents map**

Trace the file-reading/parsing loop above line 165 (where `parsed` gets populated) and add a sibling `contents := make(map[string][]byte, len(files))` populated the same way. Set `ctx.Content = contents[path]` right before the `RunAnalysisRulesWithContext` call at line 192.

- [ ] **Step 7: Build and unit-test**

```bash
go build ./...
go test ./... 2>&1 | tail -40
```

Expected: all packages pass. If any test fails, it is almost certainly one of the "MatchesASTPath" differential tests (they compare old-vs-new manually with their own ad-hoc `Content` setting, not through these call sites) — those are out of scope for this task; do not modify them here.

- [ ] **Step 8: Corpus-diff validation**

```bash
/tmp/analysis-corpus-snapshot --root test_projects/composer-src --baseline /tmp/baseline-composer.json
/tmp/analysis-corpus-snapshot --root test_projects/symfony --baseline /tmp/baseline-symfony.json
```

Expected: 0 mismatches on both (this only proves the CLI's own before/after diagnostics are identical now that it takes the CST branch — it does NOT compare against a pre-Task-1 baseline of the *ast* branch, since step 2's baseline was captured on the ast branch and this diffs the *now-CST* run against it. A 0-mismatch result here is exactly the claim the differential test suite already made per-rule: the two branches produce identical output).

- [ ] **Step 9: Commit**

```bash
git add command/analyze.go command/file_processor.go cmd/benchmark/main.go cmd/diagnostic-diff/main.go cmd/analysis-corpus-snapshot/main.go
git commit -m "analyse: wire ctx.Content into every CLI/library caller (Phase 5 step 1)"
```

---

### Task 2: Wire `AssignmentInConditionRule` and `SideEffectsRule` onto the `Content` branch

**Files:**
- Modify: `analyse/assignment_in_condition_rule.go:289-293` (registered callback)
- Modify: `analyse/side_effects_rule.go:241-244` (registered callback)
- Test: `analyse/assignment_in_condition_rule_test.go` (add a case that sets `ctx.Content` and asserts identical output to the no-Content case)
- Test: `analyse/side_effects_rule_test.go` (same)

**Interfaces:**
- Consumes: `AssignmentInConditionRule.CheckIssuesWithSource` (existing, `assignment_in_condition_rule.go:183`), `SideEffectsRule.CheckIssuesFromCST`/`CheckIssuesWithSource` (existing, `side_effects_rule.go:16,127`), `AnalysisContext.Content`.
- Produces: both rules now honor `ctx.Content` in production, matching `EmptyStatementRule`'s existing behavior. Task 6 depends on this — you cannot delete `AssignmentInConditionRule.CheckIssues`/`SideEffectsRule.CheckIssues` (the ast.Node versions) until their registered callback no longer calls them unconditionally.

- [ ] **Step 1: Write the failing test for `AssignmentInConditionRule`**

In `analyse/assignment_in_condition_rule_test.go`, add:

```go
func TestAssignmentInConditionRuleHonorsContentContext(t *testing.T) {
	src := []byte(`<?php if ($x = foo()) { bar(); }`)
	nodes, _ := syntax.ParseAST(src)
	rule := &AssignmentInConditionRule{}

	astCtx := &AnalysisContext{}
	astIssues := rule.CheckIssues(nodes, "test.php")

	cstCtx := &AnalysisContext{Content: src}
	cstIssues := runRegisteredAssignmentInConditionRule(nodes, "test.php", cstCtx)

	if len(astIssues) != len(cstIssues) {
		t.Fatalf("ast issues = %d, cst issues = %d", len(astIssues), len(cstIssues))
	}
	_ = astCtx
}
```

This test calls a helper `runRegisteredAssignmentInConditionRule` that does not exist yet — it stands in for "however the registered rule is actually invoked with a context" (check `RegisterAnalysisRule`'s registration signature in `assignment_in_condition_rule.go:289` to get the exact callback shape before writing this helper; it likely just needs you to call the same function the registration calls, but passing `cstCtx` instead of building one internally).

- [ ] **Step 2: Run it to confirm it fails to compile (helper doesn't exist) or fails on assertion**

```bash
go test ./analyse/ -run TestAssignmentInConditionRuleHonorsContentContext -v
```

- [ ] **Step 3: Modify the registered callback to branch on `Content`**

At `analyse/assignment_in_condition_rule.go:289-293`, the current registration is (confirm exact current text with `sed -n '285,295p' analyse/assignment_in_condition_rule.go` before editing, since line numbers may have shifted since the audit):

```go
RegisterAnalysisRule(..., func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	rule := &AssignmentInConditionRule{}
	return rule.CheckIssues(nodes, filename)
})
```

Change the callback body to:

```go
rule := &AssignmentInConditionRule{}
if len(ctx.Content) > 0 {
	return rule.CheckIssuesWithSource(filename, ctx.Content)
}
return rule.CheckIssues(nodes, filename)
```

(Match `CheckIssuesWithSource`'s actual signature at `assignment_in_condition_rule.go:183` — the audit did not confirm its exact parameters, verify with `sed -n '180,200p' analyse/assignment_in_condition_rule.go` first.)

- [ ] **Step 4: Repeat steps 1-3 for `SideEffectsRule`**

Same shape, using `SideEffectsRule.CheckIssuesFromCST` (`side_effects_rule.go:16`) instead of a `CheckIssuesWithSource` if that's the one that takes raw content directly (confirm which of the two — `CheckIssuesFromCST` at line 16 or `CheckIssuesWithSource` at line 127 — takes `[]byte` content vs `[]ast.Node`; the audit found both exist, use whichever matches the shape used elsewhere in this plan, i.e. takes `(filename string, content []byte)`).

- [ ] **Step 5: Run both new tests, then the full suite**

```bash
go test ./analyse/ -run 'TestAssignmentInConditionRuleHonorsContentContext|TestSideEffectsRuleHonorsContentContext' -v
go test ./... 2>&1 | tail -40
```

Expected: PASS, all green.

- [ ] **Step 6: Corpus-diff validation** (same commands as Task 1 Step 8)

- [ ] **Step 7: Commit**

```bash
git add analyse/assignment_in_condition_rule.go analyse/assignment_in_condition_rule_test.go analyse/side_effects_rule.go analyse/side_effects_rule_test.go
git commit -m "analyse: wire AssignmentInConditionRule/SideEffectsRule onto ctx.Content (Phase 5 step 2)"
```

---

### Task 3: Fix `appendReturnTypeOnNode`'s third call site

**Files:**
- Modify: `analyse/return_type_rule.go:254-274` (`ReturnTypeRule.CheckIssues` → `returnTypeIssuesForFile` → `collectReturnTypeIssues`)
- Test: `analyse/return_type_rule_test.go` (add a case exercising the fallback path with `ctx.Content` set)

**Interfaces:**
- Consumes: `CheckReturnTypeIssuesFromCST` (existing, `analyse/syntax_return_type_rule.go`), `appendReturnTypeOnNode` (existing, `analyse/return_type_rule.go:277`).
- Produces: the return-type rule's fallback path (used when `ctx.hasReturnTypeIssues` is false, i.e. `ensureStructuralIssues` hasn't run yet for this file) now also honors `ctx.Content`. Task 6 depends on this — `appendReturnTypeOnNode` cannot be deleted while this fallback still calls it unconditionally.

- [ ] **Step 1: Read the exact current code before editing**

```bash
sed -n '250,280p' analyse/return_type_rule.go
```

Confirm `collectReturnTypeIssues`'s exact signature and how it calls `appendReturnTypeOnNode` at line ~272.

- [ ] **Step 2: Write the failing test**

```go
func TestReturnTypeRuleFallbackHonorsContentContext(t *testing.T) {
	src := []byte(`<?php function f(): int { return "not an int"; }`)
	nodes, _ := syntax.ParseAST(src)
	rule := &ReturnTypeRule{}

	ctx := &AnalysisContext{Content: src, Resolver: BuildProjectIndex(map[string][]ast.Node{"test.php": nodes})}
	issues := rule.CheckIssues("test.php", nodes, ctx)
	if len(issues) == 0 {
		t.Fatal("expected at least one return-type mismatch issue")
	}
}
```

(Confirm `ReturnTypeRule.CheckIssues`'s actual parameter order/names against `return_type_rule.go:254-256` before finalizing this test — the audit gave the line range but not the exact signature.)

- [ ] **Step 3: Run it to see current behavior**

```bash
go test ./analyse/ -run TestReturnTypeRuleFallbackHonorsContentContext -v
```

If it already passes, the fallback path may already produce correct output via the ast branch (expected — this test is about the *branch taken*, not correctness of output, so also add a check that `CheckReturnTypeIssuesFromCST` specifically was the path exercised; the simplest way is a coverage-shaped assertion, or temporarily instrument, but prefer: skip a manual instrumentation hack and instead just fix Step 4 unconditionally since the audit already confirmed no `Content` branch exists there at all).

- [ ] **Step 4: Add the branch**

In `collectReturnTypeIssues` (~`return_type_rule.go:265-274`), before the existing `walkAllWithFileContext`-driven loop that calls `appendReturnTypeOnNode` at line 272, add:

```go
if len(ctx.Content) > 0 {
	return CheckReturnTypeIssuesFromCST(filename, ctx.Content, ctx)
}
```

- [ ] **Step 5: Run the test and full suite**

```bash
go test ./analyse/ -run TestReturnTypeRuleFallbackHonorsContentContext -v
go test ./... 2>&1 | tail -40
```

- [ ] **Step 6: Corpus-diff validation** (same as Task 1 Step 8)

- [ ] **Step 7: Commit**

```bash
git add analyse/return_type_rule.go analyse/return_type_rule_test.go
git commit -m "analyse: wire return-type fallback path onto ctx.Content (Phase 5 step 3)"
```

---

### Task 4: Migrate the two direct-call isolation tests

**Files:**
- Modify: `analyse/phpstan_level0_rule_test.go:1015-1037` (test calling `Level0Rule.checkClassModel`)
- Modify: `analyse/phpstan_level0_rule_test.go:1565-1585` (test calling `Level0Rule.checkSymbolsAndCalls`)

**Interfaces:**
- Consumes: `CheckClassModelIssuesFromCST` (`analyse/syntax_class_model_rule.go`), `CheckSymbolIssuesFromCST` (`analyse/syntax_symbols_rule.go`).
- Produces: no more test callers of the ast.Node-only helper methods `Level0Rule.checkClassModel`/`checkSymbolsAndCalls`. Task 6 depends on this — those two helper methods (and the raw functions they call) cannot be deleted while these tests call them.

- [ ] **Step 1: Read both tests exactly as they stand**

```bash
sed -n '1010,1040p' analyse/phpstan_level0_rule_test.go
sed -n '1560,1590p' analyse/phpstan_level0_rule_test.go
```

- [ ] **Step 2: Rewrite the first test to call the CST version**

Replace the call to `(&Level0Rule{}).checkClassModel(...)` with an equivalent call to `CheckClassModelIssuesFromCST(filename, content, ctx)`, keeping the same input PHP source and the same assertions on the returned `[]AnalysisIssue` (final-method/final-constant override diagnostics). You will need the raw source bytes for `content` — if the test currently only has `nodes []ast.Node` (parsed ahead of time), either capture the original source string used to produce those nodes (search a few lines above line 1015 for a `src := ` or `[]byte(` literal) or reparse from that same literal.

- [ ] **Step 3: Run it**

```bash
go test ./analyse/ -run TestLevel0RuleClassModel -v  # adjust to the test's actual name
```

- [ ] **Step 4: Rewrite the second test the same way, for `checkSymbolsAndCalls` → `CheckSymbolIssuesFromCST`**

Note `CheckSymbolIssuesFromCST` takes a `guards` parameter (per `analyse/syntax_fused_rule.go:37`, `CheckSymbolIssuesFromCST(filename, content, ctx, guards)`) — the old `checkSymbolsAndCalls` helper likely computes guards internally; check `analyse/phpstan_level0_symbols.go:9-15` for how it derives them and replicate that in the rewritten test (`collectReflectionGuards(nodes, ctx, fileCtx)`).

- [ ] **Step 5: Run both rewritten tests, then the full suite**

```bash
go test ./analyse/... 2>&1 | tail -40
```

- [ ] **Step 6: Commit**

```bash
git add analyse/phpstan_level0_rule_test.go
git commit -m "analyse: migrate isolation tests off ast.Node-only Level0Rule helpers (Phase 5 step 4)"
```

---

### Task 5: Rewrite the "MatchesASTPath" differential tests to drop the ast-side arm

**Files:**
- Modify: `analyse/syntax_language_rule_test.go`
- Modify: `analyse/syntax_property_callable_rule_test.go`
- Modify: `analyse/syntax_type_refs_rule_test.go`
- Modify: `analyse/syntax_symbols_rule_test.go`
- Modify: `analyse/syntax_class_model_rule_test.go`
- Modify: `analyse/syntax_method_visibility_rule_test.go`
- Modify: `analyse/syntax_throw_type_rule_test.go`
- Modify: `analyse/syntax_phpdoc_rule_test.go`
- Modify: `analyse/syntax_missing_types_rule_test.go`
- Modify: `analyse/syntax_return_type_rule_test.go`
- Modify: `analyse/syntax_fused_rule_test.go` (the whole-pipeline `TestRunAnalysisRulesWithContextCSTMatchesASTPath` test — once the ast path is gone this test's premise disappears)

**Interfaces:**
- Consumes: each file's own `Check*IssuesFromCST` function (unchanged).
- Produces: test files that assert directly on `Check*IssuesFromCST`'s output (golden expected-issue lists) instead of comparing it against the doomed ast.Node function. Task 6 depends on this for every one of these 10 files — until this task lands, Task 6 cannot delete the old functions because these tests call them by name.

**IMPORTANT — do this task file by file, running the full suite between each, not as one big edit.** These are the highest-file-count task in this plan; a mistake in one doesn't have to block the rest if you commit incrementally.

- [ ] **Step 1: Read one file fully to understand the pattern before touching any of them**

```bash
cat analyse/syntax_language_rule_test.go
```

Identify: (a) where it calls the old function (e.g. `checkLanguageOnNode` or the `Level0Rule.checkLanguage` wrapper), (b) where it calls the CST function, (c) what comparison it does between the two outputs, (d) what the actual PHP source fixtures are.

- [ ] **Step 2: For `syntax_language_rule_test.go` — remove the ast-side call and comparison, keep the CST call and turn its expected values into hardcoded golden assertions**

For each test case, replace:

```go
astIssues := checkLanguageOnNode(...) // or Level0Rule.checkLanguage(...)
cstIssues := CheckLanguageIssuesFromCST(filename, content)
if !issuesEqual(astIssues, cstIssues) { t.Fatalf(...) }
```

with:

```go
issues := CheckLanguageIssuesFromCST(filename, content)
// assert directly on `issues`: same expected codes/messages/spans the old
// comparison was implicitly relying on `astIssues` to define — read what
// astIssues would have contained for each fixture (run the test once
// BEFORE this edit with -v and capture the actual old-path output as your
// new golden values) rather than guessing expected values from scratch.
```

Concretely: before editing, run `go test ./analyse/ -run TestCheckLanguageIssuesFromCSTMatchesASTPath -v` (adjust to actual test name) with a temporary added `t.Logf("%+v", astIssues)` to capture the exact expected output per fixture, then hardcode those as the new assertions.

- [ ] **Step 3: Run this one file's tests**

```bash
go test ./analyse/ -run TestCheckLanguageIssuesFromCST -v
```

- [ ] **Step 4: Commit this file alone**

```bash
git add analyse/syntax_language_rule_test.go
git commit -m "analyse: drop ast-side comparison arm from language-rule CST test (Phase 5 step 5.1)"
```

- [ ] **Step 5: Repeat Steps 1-4 for each remaining file in this task's file list** (property-callable, type-refs, symbols, class-model, method-visibility, throw-type, phpdoc, missing-types, return-type) — one commit per file, same pattern.

- [ ] **Step 6: Handle `syntax_fused_rule_test.go` last**

`TestRunAnalysisRulesWithContextCSTMatchesASTPath` (per the audit, `analyse/syntax_fused_rule_test.go:88`) drives the whole registered-rule pipeline via `RunAnalysisRulesWithContext` itself with `ctx.Content` unset vs. set. Once Task 6 deletes the ast path, "unset" stops being a meaningful comparison arm (there will be nothing to compare against — or, if you keep `Content` optional and fall back to erroring/defaulting, decide that now). Read this test fully, then rewrite it to instead assert that `RunAnalysisRulesWithContext` with `ctx.Content` set produces the expected golden issue set for its 4 fixtures × 4 analysis levels, dropping the `astCtx`/`want` comparison arm entirely.

- [ ] **Step 7: Run full suite**

```bash
go test ./... 2>&1 | tail -60
```

- [ ] **Step 8: Commit**

```bash
git add analyse/syntax_fused_rule_test.go
git commit -m "analyse: drop ast-side comparison arm from whole-pipeline CST test (Phase 5 step 5.11)"
```

---

### Task 6: Delete the old ast.Node implementations

> **CORRECTED SCOPE (ruling made during execution, see ledger):** the original text below is WRONG about what's deletable and must not be followed literally. Investigation during implementation found the 10 `*OnNode` functions listed below (`checkLanguageOnNode`, `appendPropertyCallableTypeIssue`, `checkTypeReferenceOnNode`, `checkSymbolOnNode`, `appendClassModelOnNode`, `appendMethodVisibilityOnNode`, `appendThrowTypeOnNode`, `appendPHPDocIssuesOnNode`, `appendMissingTypeIssuesOnNode`, `appendReturnTypeOnNode`) are NOT dead — every `analyse/syntax_*_rule.go` CST-direct file lowers individual matched CST nodes back to `ast.Node` and calls these SAME functions as its own leaf-level implementation (e.g. `CheckLanguageIssuesFromCST` calls `checkLanguageOnNode` directly, per that file's own doc comment: "the existing, unmodified checkLanguageOnNode logic can run unchanged on each one"). **These 10 functions must NOT be deleted, ever, under this plan.** What IS genuinely dead and safe to delete: (1) the 4 zero-caller `Level0Rule` wrapper methods (`checkLanguage`/`checkClassModel`/`checkSymbolsAndCalls`/`checkTypeReferences`) — the fused walk never called these methods, it called the `*OnNode` functions directly; (2) the old fused-walk driver loops themselves — the `walkAllWithFileContext(...)` bodies inside `ensureSharedFileDiagnostics`, `ensureStructuralIssues`, and `collectReturnTypeIssues` — collapsed to unconditionally call their already-existing `FromCST` sibling, since `ctx.Content` is always set now (Tasks 1-3). This removes the redundant ast.Node-walk *driver* while the `*OnNode` functions survive as the CST path's own shared logic. Additionally, deleting `AssignmentInConditionRule.CheckIssues`/`SideEffectsRule.CheckIssues` (the 2 pilots' ast.Node paths) is OUT OF SCOPE for this plan — their differential tests and `*HonorsContentContext` tests deliberately still exercise the empty-`Content` fallback and were never migrated (Tasks 4-5 only covered the 10 fused-walk functions' tests). Leave both pilots' registered callbacks and `CheckIssues` methods completely untouched.

**Files:**
- Modify: `analyse/phpstan_level0_rule.go` (collapse `ensureSharedFileDiagnostics` to always call the CST path; delete the old walk body)
- Modify: `analyse/phpstan_structural_walk.go` (same collapse for `ensureStructuralIssues`)
- Modify: `analyse/phpstan_level0_language.go` (delete `checkLanguageOnNode` and `Level0Rule.checkLanguage`)
- Modify: `analyse/phpstan_level0_property_callable.go` (delete `appendPropertyCallableTypeIssue`)
- Modify: `analyse/phpstan_level0_type_refs.go` (delete `checkTypeReferenceOnNode` and its test-only helper)
- Modify: `analyse/phpstan_level0_symbols.go` (delete `checkSymbolOnNode` and `Level0Rule.checkSymbolsAndCalls`)
- Modify: `analyse/phpstan_level0_class_model.go` (delete `appendClassModelOnNode` and `Level0Rule.checkClassModel`)
- Modify: `analyse/phpstan_level2_method_visibility.go` (delete `appendMethodVisibilityOnNode`)
- Modify: `analyse/phpstan_level3_throw.go` (delete `appendThrowTypeOnNode`)
- Modify: `analyse/phpstan_level2_phpdoc.go` (delete `appendPHPDocIssuesOnNode`; keep `collectPHPDocTypeAliases` — still required by both branches per Task list Q4)
- Modify: `analyse/phpstan_level6_missing_types.go` (delete `appendMissingTypeIssuesOnNode`)
- Modify: `analyse/return_type_rule.go` (delete `appendReturnTypeOnNode`; collapse `collectReturnTypeIssues` to always call `CheckReturnTypeIssuesFromCST`)
- Modify: `analyse/assignment_in_condition_rule.go` (delete `CheckIssues`/old ast-walk body if nothing else calls it after Task 2's branch is unconditionally CST — confirm first, since `Content` might still legitimately be empty for some caller; if any caller can still reach this with empty `Content`, do NOT delete it, and instead stop here and flag this to the user rather than guessing)
- Modify: `analyse/side_effects_rule.go` (same caution as above)
- Modify: `analyse/context.go` (if `AnalysisContext.Content` should become a required, always-set field now instead of optional — read its doc comment at line 231-240 first; likely leave as-is, since a `nil`/empty `Content` may still be a legitimate "I don't have raw source" state for some exotic caller — do not remove the field or its optionality without re-confirming every caller from Task 1 truly always has bytes to give it)

**Interfaces:**
- Consumes: nothing new.
- Produces: `ensureSharedFileDiagnostics`/`ensureStructuralIssues` become thin wrappers that always call their `FromCST` counterpart; `AnalysisContext.Content` becomes a de-facto-required field even though its Go type stays `[]byte` (empty slice) for backward compatibility with any caller this plan didn't reach.

**IMPORTANT — before starting this task, re-run the Task 1 Step 2 corpus baseline capture ONE more time** (the baseline should now reflect Tasks 1-5's changes, all still on the old+new dual-path code) so you have a true pre-deletion baseline to diff against post-deletion.

- [ ] **Step 1: Re-capture the pre-deletion baseline**

```bash
cd /Users/ayan.ozturk/rg/go-php-parser
go build -o /tmp/analysis-corpus-snapshot ./cmd/analysis-corpus-snapshot
/tmp/analysis-corpus-snapshot --root test_projects/composer-src --output /tmp/pre-delete-composer.json
/tmp/analysis-corpus-snapshot --root test_projects/symfony --output /tmp/pre-delete-symfony.json
```

- [ ] **Step 2: Collapse `ensureSharedFileDiagnostics`**

Replace the entire body of `ensureSharedFileDiagnostics` (`analyse/phpstan_level0_rule.go:24-84`) with:

```go
func ensureSharedFileDiagnostics(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	ctx = ensureLevel0Context(filename, nodes, ctx)
	if ctx.hasLevel0Issues {
		return ctx
	}
	return ensureSharedFileDiagnosticsFromCST(filename, ctx.Content, nodes, ctx)
}
```

This assumes `ctx.Content` is always non-empty by this point (true after Task 1-3). If any remaining caller can reach this with empty `Content`, `ensureSharedFileDiagnosticsFromCST` will get an empty byte slice and `syntax.Parse` on it — check what that does (likely an empty/error AST) before proceeding; if it's not safe, STOP and re-audit for a missed caller rather than silently shipping broken behavior for that caller.

- [ ] **Step 3: Collapse `ensureStructuralIssues`** the same way (`analyse/phpstan_structural_walk.go:13-43`):

```go
func ensureStructuralIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	ctx = ensureSharedFileDiagnostics(filename, nodes, ctx)
	if ctx.hasStructuralIssues {
		return ctx
	}
	return ensureStructuralIssuesFromCST(filename, ctx.Content, nodes, ctx)
}
```

- [ ] **Step 4: Build — expect compile errors for now-unused old functions**

```bash
go build ./... 2>&1
```

Go does not error on unused package-level functions (only unused local vars/imports), so this may build cleanly. Use `go vet` and/or `staticcheck` (`U1000` unused-function checks) to actually find every now-dead function:

```bash
staticcheck ./analyse/... 2>&1 | grep -i "is unused"
```

- [ ] **Step 5: Delete each confirmed-unused old function, one file at a time, per the Files list above**

For each file, delete only the named old function(s). Do not delete `collectPHPDocTypeAliases`, `collectReflectionGuards`, `analysisFileTypeContext`/`CollectFileTypeContext` — these are still required by the CST path per Task list Q4 (`analyse/syntax_fused_rule.go:36-37,42-44`). After each file's deletion, run:

```bash
go build ./...
```

to catch any caller you missed.

- [ ] **Step 6: Collapse `collectReturnTypeIssues`** (`analyse/return_type_rule.go`, per Task 3's branch):

```go
func collectReturnTypeIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return CheckReturnTypeIssuesFromCST(filename, ctx.Content, ctx)
}
```

(Confirm exact current signature before replacing — re-read the file, do not assume the audit's line numbers are still exact after Tasks 1-5's edits.)

- [ ] **Step 7: For `AssignmentInConditionRule`/`SideEffectsRule` — verify before deleting**

Run:

```bash
grep -rn "AssignmentInConditionRule{}\.\?CheckIssues\b\|SideEffectsRule{}\.\?CheckIssues\b" --include="*.go" .
```

If the only remaining callers are inside the rule's own now-conditional registered callback (Task 2's `if len(ctx.Content) > 0 { ... } return rule.CheckIssues(...)` fallback), the fallback branch is *reachable* whenever `Content` is empty — do not delete `CheckIssues` while that fallback exists. Since this plan's stated goal (Global Constraints) is "every caller sets Content," you may now also simplify Task 2's branch to unconditionally call `CheckIssuesWithSource`/`CheckIssuesFromCST`, at which point `CheckIssues` (ast.Node version) becomes truly dead and deletable — do this simplification here, then delete.

- [ ] **Step 8: Full build, vet, test**

```bash
go build ./...
go vet ./...
go test ./... 2>&1 | tail -60
```

- [ ] **Step 9: Corpus-diff against the pre-deletion baseline**

```bash
/tmp/analysis-corpus-snapshot --root test_projects/composer-src --baseline /tmp/pre-delete-composer.json
/tmp/analysis-corpus-snapshot --root test_projects/symfony --baseline /tmp/pre-delete-symfony.json
```

Expected: 0 mismatches. Any mismatch means a code path this plan didn't account for was still using the ast branch and now silently produces different output — do not proceed to commit if there is any mismatch; stop and investigate.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "analyse: delete dead ast.Node implementations of 13 CST-ported rules (Phase 5 step 6)"
```

---

### Task 7: Validate against vscode-php-strom (cross-repo safety net)

**Files:** none in go-php-parser; this task only runs commands in the sibling repo.

**Interfaces:**
- Consumes: the finished, committed state of go-php-parser from Task 6.
- Produces: proof the sibling repo's pinned build and its own test suite still pass against these deletions once its pin is bumped again.

- [ ] **Step 1: Push go-php-parser's Task 1-6 commits to GitHub**

This is a push to a shared remote — confirm with the user before running, per this session's established operational rule (the same rule that gated the Phase 4 push).

```bash
cd /Users/ayan.ozturk/rg/go-php-parser
git push origin main
```

- [ ] **Step 2: Bump vscode-php-strom's pin**

```bash
cd /Users/ayan.ozturk/rg/vscode-php-strom/server
GOWORK=off go get github.com/ayanozturk/go-php-parser@$(git -C /Users/ayan.ozturk/rg/go-php-parser rev-parse origin/main)
GOWORK=off go mod tidy
```

- [ ] **Step 3: Validate both build modes**

```bash
cd /Users/ayan.ozturk/rg/vscode-php-strom
make test-server-dev
make test-server
```

Expected: all packages pass, identical to the Phase 4 validation already done in this session.

- [ ] **Step 4: Commit the pin bump**

```bash
cd /Users/ayan.ozturk/rg/vscode-php-strom
git add server/go.mod server/go.sum
git commit -m "server: bump go-php-parser pin (Phase 5 ast.Node cleanup)"
```

---

### Task 8: Update `AGENTS.md` to reflect true Phase 5 completion

**Files:**
- Modify: `/Users/ayan.ozturk/rg/go-php-parser/AGENTS.md` (the "CST-direct migration" section, `:343-344` — the old "Phase 5 remains unstarted" line — plus anywhere describing the 13 rules as fully production-wired, which this plan's own audit showed was only true for 11 of 13 before this plan ran)

**Interfaces:** none — documentation only.

- [ ] **Step 1: Replace the "Phase 5 remains unstarted" paragraph**

Find (per this session's earlier edit) the paragraph ending "...Phase 5 (cleanup — removing the now-unused `ast.Node`/lowering path for the 13 ported rules) remains unstarted and requires explicit user confirmation before starting..." and replace it with a summary of what Tasks 1-7 actually did: every caller now sets `ctx.Content`, the two dormant pilots (`AssignmentInConditionRule`/`SideEffectsRule`) were wired in, the `appendReturnTypeOnNode` third call site was fixed, and the 10+3 old ast.Node implementations were deleted, corpus-validated to 0 mismatches on composer-src/symfony both before and after.

- [ ] **Step 2: Commit**

```bash
cd /Users/ayan.ozturk/rg/go-php-parser
git add AGENTS.md
git commit -m "AGENTS.md: record Phase 5 (ast.Node cleanup) completion"
```

- [ ] **Step 3: Ask the user whether to push this final documentation commit** (and Task 6's commits, if Task 7 wasn't run yet) to GitHub.

---

## Self-Review Notes

- **Spec coverage:** every gap the audit found (Q1-Q7 in the scoping investigation) maps to a task: caller wiring → Task 1; dormant pilots → Task 2; return-type third call site → Task 3; isolation tests → Task 4; differential test rewrites → Task 5; actual deletion → Task 6; cross-repo → Task 7; docs → Task 8.
- **Placeholder scan:** Task 5's per-file steps intentionally describe a repeatable *procedure* rather than writing out all 10 files' exact old/new assertions inline, because the audit did not capture each fixture's literal expected-issue text — Step 2 of Task 5 tells the executor exactly how to derive real values (run the old path once, capture its output, hardcode it) rather than inventing them. This is a deliberate, disclosed exception to "no placeholders," not an oversight.
- **Type consistency:** `ctx.Content []byte` is used identically across all tasks; `CheckClassModelIssuesFromCST`/`CheckSymbolIssuesFromCST`/etc. signatures are taken from Task list Q1's audit (`analyse/syntax_fused_rule.go:47-63`) and referenced consistently in Tasks 2-6.
- **Known open risk carried forward from the original AGENTS.md text (not resolved by this plan):** the CST-direct branch currently reparses the file once per `Check*IssuesFromCST` call (up to ~11 times per file) rather than sharing one `syntax.Walk`. This plan does not fix that — it was explicitly deferred in the original Phase 4 work as a separate follow-up, and Task 6's deletion makes the CST path the *only* path everywhere, so this reparse cost is now permanent, unconditional production cost, not opt-in. Flag this to the user as a strong candidate for a follow-up plan after this one ships.
