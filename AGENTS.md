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

### Perf (not coverage %)

- Bench gates: allocs/op, tokens/KB on Symfony/WordPress-sized inputs — regressions fail CI even if coverage is high.
- Corpus identity + metrics: `SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m` (same for `wordpress-develop`); JSON reports via `go run ./cmd/syntax-metrics --root test_projects/symfony --json`.

### Practical CI target

- New `syntax` / trivia / binder packages: **≥95%** (or 100% if critical-unit-testing bar applies to those packages only).
- Whole-module line coverage: **non-goal** during cutover; round-trip + gold + analyse suite are the merge bar.

**Summary:** 100% on the identity/gold contract, very high on the new kernel, corpus/analyse green for integration — not “100% of go-php-parser.”
