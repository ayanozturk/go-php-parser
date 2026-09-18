# Codex Instructions

This repo is the Go PHP parser and static analyser. Sibling LSP delivery lives in `vscode-php-strom` (`server/go.mod` pins this parser).

## Testing coverage (lossless parser cutover)

Do **not** chase a single line-% on the whole repo. Use layered targets.

### Must-hold (correctness gates)

- **100% lossless invariant** on fixtures + corpus: `Print(Parse*(src)) == src` for every file that lexes (parse-error cases round-trip via missing/error tokens).
- **Gold trees** for lossy hotspots: attributes, DNF types, names (qualified/relative/FQ), heredoc/nowdoc + interpolation, mixed-case keywords, trivia attachment on decls.
- **Token parity** fixtures vs `token_get_all` (kind + text) for Zend edge cases — small set, high value.

### Package-level (unit)

- `syntax` + new binder paths: aim very high (≈90–100% of *new* code).
- Lexer state machines (attribute/`#[`, encapsed, heredoc): branch-complete on new states, not “lines in lexer.go”.
- Legacy `ast` / old string parsers: coverage may fall as code is deleted — don’t maintain 100% on code being removed.

### Analyse / Strom

- Prefer **behavioral coverage**: binding, rename/refs, format identity, type-from-syntax — not line % on giant rule files.
- Keep differential/level fixtures green; add cases when string→syntax edges move.

### CST-direct migration (retiring the ast.Node double-representation)

`analyse/` mostly walks `[]ast.Node` lowered from the `syntax` CST
(`syntax/lower`), which is an intentional interim state, not the target
architecture — the goal is for rules to walk `*syntax.RedNode` directly and
drop the lowering pass for the analysis path. `analyse/syntax_binder.go` is
the original CST-native template; `syntax.RedNode.Pos()`/`EndPos()` and
`syntax.Walk` are the shared contract for new CST-native rules (see
`analyse/empty_statement_rule.go` for the first migrated rule). Migrate rules
incrementally behind the differential/gold suites — do not attempt a bulk
rewrite of `analyse/phpstan_level0_walk.go`'s dispatcher in one pass.

`syntax/conditions.go` provides reusable, unit-tested condition and body
accessors (`IfCondition`, `WhileCondition`, `DoWhileCondition`,
`ForConditions`, `MatchCondition`, `MatchArmConditions`, `IfBody`,
`IfElseIfs`, `IfElse`, `ElseIfBody`, `ElseBody`, `WhileBody`, `DoWhileBody`,
`ForBody`, `StatementBodyList`, `FunctionBody`, `ClassMethods`,
`CallIsMethodLike`, `CallArgList`) so new rule migrations don't need to
re-derive child-detection logic from `syntax/lower_stmt.go`/`lower_expr.go`.
`analyse/assignment_in_condition_rule.go`'s `CheckIssuesWithSource` is the
second migrated rule (test-isolation path only; the registered rule still
uses the ast.Node-based `CheckIssues`) and is the template for rules that
recurse into *expressions*, not just statement bodies: a flat `syntax.Walk`
from the root does NOT preserve the statement-vs-expression traversal
boundary that hand-written `ast.Node` type switches get for free (e.g. it
would also visit a `KindMatchExpr` nested inside another statement's
condition as if it were top-level). The fix is a bespoke recursive
statement walker that only descends into known body-container kinds, and
calls a separate, narrowly-scoped expression walker on isolated condition
subtrees only. See `/memories/repo/cst-direct-migration.md` for the full
design writeup and the list of sharp edges found (CallExpr method-vs-
function distinction, transparent ParenExpr unwrapping, KindArg/KindArgList
wrapper nodes around call arguments) before porting another rule.

`analyse/side_effects_rule.go`'s `CheckIssuesFromCST` is the third migrated
rule and the last one sharing the plain `RegisterAnalysisRule` +
self-contained `CheckIssuesWithSource` shape — every remaining rule is
registered via `RegisterAnalysisRuleWithContext`/`WithLevel`/`WithMeta` and
fed by the `phpstan_level0_walk.go` fused dispatcher, so further rule-by-rule
ports are blocked until that dispatcher (or `RunAnalysisRulesWithContext`'s
`[]ast.Node` signature) is migrated. Added `syntax.NamespaceBody` and
`syntax.ExpressionStmtExpr` accessors for this port; see
`/memories/repo/cst-direct-migration.md` for the `KindToken`-exclusion
pitfall (a `KindFile`'s children always include a trailing EOF token) and
the `declare(){}` quirk this rule deliberately preserves.

The dispatcher/signature migration itself is being done as an approved,
user-gated 5-phase plan (Phase 0: differential corpus harness; Phase 1: CST
walker skeleton; Phase 2: CST `FileTypeContext`; Phase 3: per-rule ports;
Phase 4: flip the public signature + the cross-repo call site in
`vscode-php-strom/server/providers/diagnostics.go`; Phase 5: cleanup) — do
not skip ahead to a later phase without explicit confirmation, since Phase 4
touches the production LSP diagnostics entry point in the sibling repo.
Phases 0 and 1 are done: `analyse/syntax_walk.go`'s `walkSyntaxConfigured` is
an additive, not-yet-wired-in CST analogue of `walkAllConfigured` — and,
unlike the ast.Node version, it does NOT need 43 type-switch cases, because
`*syntax.RedNode.Children()` is uniform across all kinds (ast.Node isn't).
It only needs a small "pure container kind" exclusion set
(`syntaxContainerOnlyKinds`) and class/function scope-boundary tracking.
`cmd/analysis-corpus-snapshot` is the Phase 3/4 safety net: it runs the full
rule registry over every `.php` file under `--root` and can capture/diff a
JSON snapshot of every file's issue set via `--output`/`--baseline`. Phase 2
is also done: `analyse/syntax_file_type_context.go`'s
`CollectFileTypeContextFromSyntax` is the CST-direct analogue of
`CollectFileTypeContext`, reusing `syntax.NamespaceBody`,
`syntax.AppendTypedUseAliases`, and 3 new thin exported wrappers
(`syntax.IsNameKind`, `syntax.ClauseNames`, `syntax.UnqualifiedTail`) rather
than duplicating name-classification logic; it deliberately leaves
`ClassNodes`/`Constants` unpopulated (documented, narrow-use gaps — not bugs)
and was corpus-validated with zero mismatches across composer-src (1006
files) and symfony (10467 files) in addition to its unit fixtures. See
`/memories/repo/cst-direct-migration.md` for full design notes and gotchas
(e.g. `RedNode.Text()` on an assignment expression excludes the trailing
statement semicolon).

Phase 3 (per-rule ports) has started: `analyse/syntax_language_rule.go`'s
`CheckLanguageIssuesFromCST` is the CST-direct analogue of
`checkLanguageOnNode` (goto/label, duplicate array keys, include/require
existence, non-writable ++/--, void/unset casts, invalid regex, printf/
sprintf placeholder counts). It was picked first because it ignores
`FileTypeContext`/class/currentFn entirely, so a flat `syntax.Walk` is safe
and it needed no `walkSyntaxConfigured` wiring. New reusable
`syntax.LowerExprNode`/`syntax.LowerStmtNode` wrappers lower *just the
matched CST subtree* to `ast.Node` so the existing unmodified
`checkLanguageOnNode` (and its `issueSpan` helper) can run unchanged on it —
avoids re-deriving literal-extraction logic against raw CST tokens.
**Corpus validation surfaced a real, pre-existing gap in `walkAllConfigured`
itself:** it has no case for `*ast.SwitchNode`/`*ast.SwitchCaseNode`, so
switch bodies (conditions and case bodies alike) are invisible to every rule
it drives, not just this one. The CST port deliberately preserves this (skips
`KindSwitchStmt` subtrees) to stay parity-safe; it does not fix the
underlying gap. See `/memories/repo/cst-direct-migration.md` for the full
writeup before porting the next callback (`checkTypeReferenceOnNode`/
`checkSymbolOnNode`/etc. all need real `FileTypeContext` scope, unlike this
one, and should double-check against the same switch-body gap).

Second Phase 3 port: `analyse/syntax_property_callable_rule.go`'s
`CheckPropertyCallableTypeIssuesFromCST` is the CST-direct analogue of
`appendPropertyCallableTypeIssue` (typed properties/promoted constructor
params can't declare `callable`). Also context-free, so still no
`walkSyntaxConfigured` wiring needed. Added `syntax.LowerPropertyDeclNode`/
`syntax.LowerParamNode` wrappers (property/param declarations aren't
statements or expressions, so the existing `LowerExprNode`/`LowerStmtNode`
don't cover them). Unlike the language-rule port, this walk must NOT skip
descending into a matched `KindPropertyDecl`/`KindParam`'s children —
default values can contain nested closures with their own promoted-looking
params, and the ast-side walk visits those. Corpus-validated with zero
mismatches on the first run (composer-src + symfony). See
`/memories/repo/cst-direct-migration.md` for the full writeup.

Third Phase 3 port: `analyse/syntax_type_refs_rule.go`'s
`CheckTypeReferenceIssuesFromCST` is the CST-direct analogue of
`checkTypeReferenceOnNode` (unresolved use-function/const imports, param/
return/property/constant type references, caught-class/non-throwable
checks, unresolved attribute classes). First port needing real
`FileTypeContext`/`ctx.Resolver`, so first to drive traversal via
`walkSyntaxConfigured` + `CollectFileTypeContextFromSyntax` together. This
port's corpus validation (composer-src + symfony, run repeatedly across two
sessions) surfaced by far the most gaps so far, ALL now centralized in
`walkSyntaxConfigured` itself (not per-rule) since every one is either a
genuine `walkAllConfigured`/lowering-pipeline blindness or a reusable
signal: (1) `lowerStmt` has no case for declaration kinds at all — a class/
function/etc. declared inside ANY statement body is dropped from the
`ast.Node` model entirely, not just unwalked (core lowering-pipeline gap,
not a dispatcher gap); (2) anonymous-class member lists are exempt from (1)
since they lower via `lowerClassMembers`; (3) `*ast.YieldNode` has no
dispatcher case at all; (4) enum cases (and their own preceding attribute
sibling) are invisible, same as (5) class-constant attributes (same
structural shape as the pre-existing class-constants gap); (6)
`*ast.FirstClassCallableNode` (`foo(...)` syntax) has no dispatcher case at
all, hiding e.g. `(new class {...})->m(...)`; (7) a *new* `inStatementBody`
signal was added to `walkSyntaxConfigured`'s public callback signature
(gates attributes floating before statement-body expressions, e.g. `return
#[Attr] function(){};`) — but had to be broadened (any `syntax.
IsStatementKind` node, not just `KindStatementList` bodies) then narrowed
again (reset to false inside `KindParam`, since param attributes are always
visible ast-side regardless of nesting depth via `FunctionNode.Params`).
Corpus-validated to **0 mismatches** on both composer-src (1006 files) and
symfony (10478 files) after all fixes. See
`/memories/repo/cst-direct-migration.md` for the full writeup, including
the debugging techniques that found each gap (before porting the next
callback: `checkSymbolOnNode`/`appendClassModelOnNode`/etc.).

Fourth Phase 3 port: `analyse/syntax_symbols_rule.go`'s
`CheckSymbolIssuesFromCST` is the CST-direct analogue of `checkSymbolOnNode`
(unresolved instantiations, function/method/static calls, constant/property
fetches). First port needing a "mark now, check later" tracking map
(`callCallees`, to avoid double-reporting a method-call receiver as a
standalone property fetch) — discovered `syntax.RedNode.Children()`
allocates a fresh wrapper on every call (no caching), so such maps must be
keyed by `(Green, Offset)` pairs, not raw `*RedNode` pointers. Corpus
validation (composer-src + symfony) surfaced four more `walkAllConfigured`/
lowering-pipeline gaps beyond the type-refs port's list, all centralized in
`walkSyntaxConfigured`: (1) `*ast.TypeCastNode` has no dispatcher case at
all, hiding a cast's operand; (2) `lowerTernaryExpr` aliases `IfTrue` to the
same node as `Condition` for Elvis (`cond ?: else`) form, so
`walkAllConfigured` visits/reports that shared subtree twice — replicated
via an extra walk, not fixed; (3) `*ast.ParamNode`'s dispatcher case never
walks `n.DefaultValue`, hiding param default value expressions; (4)
`*ast.ClassNode` never walks `n.Constants` at all, hiding a class constant's
whole value expression (not just its attributes, as the type-refs port had
already found). Two more fixes were local to this rule, not the shared
walker: a `suppressed`-subtree marking mechanism for when `lowerCallExpr`
silently drops an entire dynamic-callee call (`$obj->{$expr}()`) including
nested args; and an anonymous-class constructor-args scope bug where `new
class($this->x) {...}`'s ctor args were being checked against the anonymous
class's own scope instead of the enclosing scope. Also found: PHP 8.4
property hook bodies (`get {}`/`set {}`) are entirely invisible to the
Level0 ast dispatcher, so `KindPropertyHookList` subtrees must be skipped
entirely to avoid CST-side over-detection. Corpus-validated to **0
mismatches** on both composer-src (1006 files) and symfony (10467 files).
See `/memories/repo/cst-direct-migration.md` for the full writeup (before
porting the next callback: `appendClassModelOnNode`/etc.).

Fifth Phase 3 port: `analyse/syntax_class_model_rule.go`'s
`CheckClassModelIssuesFromCST` is the CST-direct analogue of
`checkClassModel`/`appendClassModelOnNode` (final+abstract conflicts,
extends/implements legality, method/constant/constructor legality,
interface member visibility, readonly property overrides, enum legality,
trait-use resolution). Unlike every prior context-dependent port, this one
needed **zero** reimplementation of check logic: `appendClassModelOnNode`
only fires on 4 already-fully-lowered declaration types
(`*ast.ClassNode`/`*ast.InterfaceNode`/`*ast.TraitUseNode`/`*ast.EnumNode`),
so the port just finds the right CST node once per declaration and lowers
that one subtree via three new thin wrappers added to `syntax/api.go`
(`LowerInterfaceDeclNode`/`LowerEnumDeclNode`/`LowerTraitDeclNode` — needed
because the pre-existing `LowerClassLikeContextNode` deliberately returns
only a synthetic Name-only stand-in for interfaces/traits/enums, since it
mirrors `walkAllConfigured`'s "class" *context* parameter, not a general
lowering utility). `*ast.TraitUseNode` isn't a CST declaration kind in its
own right — it's extracted from the already-lowered `ClassNode.Properties`/
`TraitNode.Body` slices it gets prepended into by `lowerClassMembers`/
`lowerTraitMembers`, mirroring exactly how `walkAllConfigured` reaches it
too. Corpus-validated to **0 mismatches on the first run** on both
composer-src (532 non-vendor targets) and symfony (10016 non-vendor
targets) — no new `walkSyntaxConfigured` gaps found. See
`/memories/repo/cst-direct-migration.md` for the full writeup (before
porting the next candidate: the structural pass in
`phpstan_structural_walk.go`).

Sixth Phase 3 port: four CST-direct rule files covering the structural
pass — `analyse/syntax_method_visibility_rule.go`'s
`CheckMethodVisibilityIssuesFromCST` (→ `appendMethodVisibilityOnNode`),
`analyse/syntax_throw_type_rule.go`'s `CheckThrowTypeIssuesFromCST` (→
`appendThrowTypeOnNode`), `analyse/syntax_phpdoc_rule.go`'s
`CheckPHPDocIssuesFromCST` (→ `appendPHPDocIssuesOnNode`), and
`analyse/syntax_missing_types_rule.go`'s `CheckMissingTypeIssuesFromCST` (→
`appendMissingTypeIssuesOnNode`) — all reusing the same
`walkSyntaxConfigured`-driven dispatch shape as prior ports. Symfony corpus
validation surfaced one real (not preserved-gap) CST-side bug: a property's
PHPDoc nested inside an anonymous class reached via `(new class
{...})::class` was over-detected as referencing an unknown class for
`self::*` syntax, because `splitStaticMemberAccessParts`
(`syntax/lower_expr.go`) — shared by both `lowerStaticMemberAccessExpr` and
`lowerCallExpr`'s static-call-callee branch — discards the entire lowered
ast.Node for a non-Name class-part expression before `::`, keeping only its
`.TokenLiteral()` string; ast-side can therefore never reach that anonymous
class's members at all (0 issues is correct there), while the CST walker
was still descending into it. Fixed by centralizing a new
`staticMemberAccessDynamicClassPart` skip in `analyse/syntax_walk.go` (same
"no ast.Node wrapper exists" shape as the pre-existing `ParamDefaultValue`
skip) rather than in the phpdoc rule itself, since it affects every rule
driven by the shared walker. Corpus-validated to **0 mismatches** on both
composer-src (532 files) and symfony (10016 files) after the fix. See
`/memories/repo/cst-direct-migration.md` for the full writeup (before
porting the last candidate: `appendReturnTypeOnNode`).

Seventh and final Phase 3 port: `analyse/syntax_return_type_rule.go`'s
`CheckReturnTypeIssuesFromCST` is the CST-direct analogue of
`appendReturnTypeOnNode` (declared-vs-actual return type mismatches, void/
never-function issues, return-path completeness). Dispatches on
`KindFunctionDecl`/`KindMethodDecl` (interface methods lowered via
`LowerInterfaceMethodDeclNode` pass through as a harmless no-op, since
`appendReturnTypeOnNode` only ever matches `*ast.FunctionNode` via a type
assertion — exactly mirroring ast-side production behavior) and
`KindClosureExpr`. De-risked before implementation rather than via a
corpus mismatch this time: `FlowScopeKeyForNode`/`flowScopeKey`
(`analyse/control_flow.go`) is purely position-based (filename + offsets),
not pointer-identity based, so CST-lowered function nodes still correctly
match pre-populated `ctx.Flow` scope entries; and
`analysisFunctionScope`'s cache (`ctx.functionScopeByNode`) is keyed by
`*ast.FunctionNode` pointer, so reusing one `*AnalysisContext` for both the
ast-side and cst-side calls in the parity test is safe. Corpus-validated
to **0 mismatches** on both composer-src (532 files) and symfony (10027
files) on the first run — no new `walkSyntaxConfigured` gaps found despite
this being the largest/riskiest remaining candidate. **This completes
Phase 3** — all 7 identified rule-port candidates are now ported,
unit-tested, and corpus-clean. See `/memories/repo/cst-direct-migration.md`
for the full writeup.

Phase 4 is done, but with a materially narrower/safer shape than the
original plan's wording ("flip `RunAnalysisRulesWithContext`'s signature").
Investigating the full registered-rule set before implementing revealed
Phase 3 only ported the 10 checks inside `Level0Rule`'s fused ast.Node walk
(`ensureSharedFileDiagnostics` in `phpstan_level0_rule.go`) plus 3 earlier
pilots (EmptyStatementRule, AssignmentInConditionRule, SideEffectsRule) —
about 13 of ~28 total registered rules. A hard signature flip (dropping
`[]ast.Node` from `RunAnalysisRulesWithContext`) would have broken roughly
15 other genuinely-separate, still-ast.Node-only registered rules (arg
count/type, deprecated calls, level1 variables, level2 method existence/
non-object, level6/7/8 method checks, property type, unreachable code,
etc.), none of which were ever in scope for CST-direct porting.

Instead, `RunAnalysisRulesWithContext`'s public signature is **unchanged**.
`AnalysisContext` gained an optional `Content []byte` field (`context.go`);
when a caller sets it, `ensureSharedFileDiagnostics`
(`phpstan_level0_rule.go`) and `ensureStructuralIssues`
(`phpstan_structural_walk.go`) — the two shared per-file cache-populating
funnel functions that every one of the ~13 fused-walk rules is a thin
wrapper around — branch to a new CST-direct fused path
(`ensureSharedFileDiagnosticsFromCST`/`ensureStructuralIssuesFromCST` in
new file `analyse/syntax_fused_rule.go`) instead of the `[]ast.Node`
`walkAllWithFileContext` walk, calling the already-ported/validated 10
`Check*IssuesFromCST` functions plus a new small
`checkEmptyStatementIssuesFromCST` helper (mirroring
`EmptyStatementRule.CheckIssuesWithSource`'s existing raw-content branch).
Left unset (the default for every existing caller/test), behavior is
byte-for-byte identical to before this field existed — this was an
additive opt-in, not a breaking migration. `guards`
(`collectReflectionGuards`) and `ctx.phpDocTypeAliases` are still derived
from `nodes` (no CST-only equivalent exists for either), so `nodes` remains
a required parameter on both funnel functions even on the CST-direct
branch.

Validation: a new `TestRunAnalysisRulesWithContextCSTMatchesASTPath`
(`analyse/syntax_fused_rule_test.go`) drives the *whole* registered-rule
pipeline via `RunAnalysisRulesWithContext` itself (not one rule in
isolation) with `ctx.Content` unset vs. set, across 4 fixtures × 4
analysis levels (nil/0/2/6) — proving the swap is invisible end-to-end,
not just per-rule. A throwaway corpus-diff harness (deleted after use, per
its own header comment) additionally corpus-validated the full pipeline to
**0 mismatches** on composer-src (532 files) and symfony (10027 files, 177s
runtime). The re-parse-cost question (each of the 10 `Check*IssuesFromCST`
functions independently called `syntax.Parse`/`syntax.ParseAST`, so the
CST-direct branch reparsed the file up to ~11 times instead of the
ast.Node path's single walk) has since been fixed: the 10 `Check*IssuesFromCST`
functions were split into thin `Check*IssuesFromCST` wrappers plus
`check*IssuesFromParsed(filename string, res *syntax.ParseResult, ...)`
counterparts, and both `ensureSharedFileDiagnosticsFromCST` and
`ensureStructuralIssuesFromCST` (`analyse/syntax_fused_rule.go`) now call
`syntax.Parse` exactly once per file and pass the shared `*syntax.ParseResult`
to every `*FromParsed` call. A new benchmark
(`BenchmarkEnsureSharedFileDiagnosticsFromCST`,
`analyse/syntax_fused_rule_bench_test.go`) measured the real effect on a
fixture exercising all 10 check families at analysis level 6: before the fix,
~3.25ms/op, ~2.44MB/op, ~38,578 allocs/op; after, ~2.06ms/op, ~1.21MB/op,
~26,545 allocs/op — roughly a 1.6x speedup on `ns/op` (and ~1.45x fewer
allocations), notably below the naively-expected ~11x from parse-count alone
because `syntax.Parse` is not the dominant cost relative to the checks' own
tree walks. Numbers were captured via `go test ./analyse/ -bench
BenchmarkEnsureSharedFileDiagnosticsFromCST -benchtime=3x -benchmem -run '^$'`
on both the current tree and a temporary revert to the pre-fix shape
(`git checkout 33b6ba4b -- analyse/syntax_fused_rule.go
analyse/syntax_class_model_rule.go analyse/syntax_language_rule.go
analyse/syntax_method_visibility_rule.go analyse/syntax_missing_types_rule.go
analyse/syntax_phpdoc_rule.go analyse/syntax_property_callable_rule.go
analyse/syntax_return_type_rule.go analyse/syntax_symbols_rule.go
analyse/syntax_throw_type_rule.go analyse/syntax_type_refs_rule.go`, restored
afterward).

Cross-repo: with explicit user confirmation, go-php-parser's local `main`
(through `33cd38f7`) was pushed to GitHub, `vscode-php-strom/server/go.mod`'s
pin was bumped to that commit (`go get .../go-php-parser@33cd38f7...` +
`go mod tidy`), and `runAnalysisRulesForSource`
(`server/providers/diagnostics.go`) now sets `ctx.Content = source` right
before calling `RunAnalysisRulesWithContext` (raw source bytes were already
computed there for `sharedcache.StoreCachedFileContent`). Validated via both
`make test-server-dev` (sibling checkout) and `make test-server`
(`GOWORK=off` against the bumped pin) — all packages pass. **This completes
Phase 4 in production**, not just as an opt-in library capability.

Phase 5 (cleanup) is done, but landed materially narrower than originally
scoped — the plan that kicked it off
(`docs/superpowers/plans/2026-09-17-cst-direct-phase5-cleanup.md`) assumed
the 13 ported rules' old `ast.Node` implementations were dead code once
every caller set `ctx.Content`. Two rounds of investigation during
execution found that assumption wrong in ways worth recording so nobody
re-attempts the original framing:

1. **The 10 `*OnNode`/`append*OnNode` functions (`checkLanguageOnNode`,
   `appendPropertyCallableTypeIssue`, `checkTypeReferenceOnNode`,
   `checkSymbolOnNode`, `appendClassModelOnNode`,
   `appendMethodVisibilityOnNode`, `appendThrowTypeOnNode`,
   `appendPHPDocIssuesOnNode`, `appendMissingTypeIssuesOnNode`,
   `appendReturnTypeOnNode`) are not dead — they never were the "old ast
   path" to retire.** Every `analyse/syntax_*_rule.go` CST-direct file
   lowers its individual matched CST nodes back to `ast.Node` (via
   `syntax.LowerStmtNode`/`syntax.LowerExprNode`) and calls these same,
   unmodified functions as its own leaf-level implementation — see
   `analyse/syntax_language_rule.go`'s doc comment, which says so
   explicitly ("the existing, unmodified checkLanguageOnNode logic can run
   unchanged on each one"). These functions are permanent shared logic
   between both the CST walker and the (now-removed) old fused walk, and
   must not be deleted.
2. **What actually was dead and got removed:** the 4 zero-caller
   `Level0Rule` wrapper methods (`checkLanguage`, `checkClassModel`,
   `checkSymbolsAndCalls`, `checkTypeReferences` — the fused walk never
   called these, it called the `*OnNode` functions directly), and the 3 old
   fused-walk *driver* loops themselves: `ensureSharedFileDiagnostics`
   (`phpstan_level0_rule.go`), `ensureStructuralIssues`
   (`phpstan_structural_walk.go`), and `collectReturnTypeIssues`
   (`return_type_rule.go`) each now unconditionally call their existing
   `FromCST` sibling instead of branching on `ctx.Content`.
3. **A real caller was still reaching these with empty `Content`**:
   `command/file_processor.go`'s `runAnalysis` had an early-return special
   case (`configuredAnalysisLevel == nil && project == nil`, the `style`
   command's default/no-`--level` invocation) that called the nil-context
   `analyse.RunAnalysisRules` wrapper, never populating `Content`. Fixing
   this naively (folding it into the general branch) introduced a second,
   subtler bug — it also started eagerly building `ctx.Resolver` where the
   old path left it `nil`, silently activating resolver-dependent rules
   (`A.DEPRECATED.CALL`, `A.ARG.TYPE`, cross-object `A.PROP.TYPE`, etc.)
   that read `ctx.Resolver` without building it and had relied on it
   staying unset. The corpus-diff tool couldn't catch this (it always sets
   both `Content` and a resolver). Fixed by keeping the early-return
   special case but changing what it calls to
   `RunAnalysisRulesWithContext(path, nodes, &analyse.AnalysisContext{Content: content})`
   — `Resolver` stays nil exactly as before, only `Content` is new. Locked
   with `command/nolevel_style_path_test.go`'s
   `TestNoLevelStylePathSuppressesResolverDependentRules`, which was
   verified adversarially (fails under the buggy merged form, passes under
   the fix).
4. **The 2 pilot rules' (`AssignmentInConditionRule`, `SideEffectsRule`)
   `ast.Node` `CheckIssues` methods and their differential/
   `*HonorsContentContext` tests were left in place** — only their
   registered callbacks were wired onto `ctx.Content` (in an earlier task
   in this same plan, via `RegisterAnalysisRuleWithContext` and new
   `runRegistered*Rule` functions), not their underlying `CheckIssues`
   implementations; deleting those `ast.Node` paths is out of scope for
   this cleanup and remains genuinely future work if anyone wants full
   CST-only cutover for those two.

Corpus-diff (composer-src 532 files, symfony 10016 files) is 0 mismatches
against pre-cleanup baselines throughout. The still-unaddressed reparse
cost (each of the 10 `Check*IssuesFromCST` functions independently calls
`syntax.Parse`) remains live on every diagnostics run and is unaffected by
this cleanup — worth remeasuring as a separate follow-up if it matters in
practice.

### Perf (not coverage %)

- Bench gates: allocs/op, tokens/KB on Symfony/WordPress-sized inputs — regressions fail CI even if coverage is high.
- Corpus identity + metrics: `SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m` (same for `wordpress-develop`); JSON reports via `go run ./cmd/syntax-metrics --root test_projects/symfony --json`.

### Practical CI target

- New `syntax` / trivia / binder packages: **≥95%** (or 100% if critical-unit-testing bar applies to those packages only).
- Whole-module line coverage: **non-goal** during cutover; round-trip + gold + analyse suite are the merge bar.

**Summary:** 100% on the identity/gold contract, very high on the new kernel, corpus/analyse green for integration — not “100% of go-php-parser.”
