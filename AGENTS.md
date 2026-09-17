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

### Perf (not coverage %)

- Bench gates: allocs/op, tokens/KB on Symfony/WordPress-sized inputs — regressions fail CI even if coverage is high.
- Corpus identity + metrics: `SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m` (same for `wordpress-develop`); JSON reports via `go run ./cmd/syntax-metrics --root test_projects/symfony --json`.

### Practical CI target

- New `syntax` / trivia / binder packages: **≥95%** (or 100% if critical-unit-testing bar applies to those packages only).
- Whole-module line coverage: **non-goal** during cutover; round-trip + gold + analyse suite are the merge bar.

**Summary:** 100% on the identity/gold contract, very high on the new kernel, corpus/analyse green for integration — not “100% of go-php-parser.”
