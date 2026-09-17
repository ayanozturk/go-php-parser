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

`syntax/conditions.go` provides reusable, unit-tested condition accessors
(`IfCondition`, `WhileCondition`, `DoWhileCondition`, `ForConditions`,
`MatchCondition`) so new rule migrations don't need to re-derive condition
child-detection logic from `syntax/lower_stmt.go`. Note: a fully faithful
CST port of a rule that recurses into *expressions* (not just statement
bodies) is trickier than it looks — a flat `syntax.Walk` from the root
doesn't preserve a statement-vs-expression traversal boundary the way
hand-written `ast.Node` type switches do (e.g. it will also visit a
`KindMatchExpr` nested inside another statement's condition, which the
original rule wouldn't). See `/memories/repo/cst-direct-migration.md` for the
full writeup (from investigating `assignment_in_condition_rule.go`) before
attempting another rule with nested expression recursion.

### Perf (not coverage %)

- Bench gates: allocs/op, tokens/KB on Symfony/WordPress-sized inputs — regressions fail CI even if coverage is high.
- Corpus identity + metrics: `SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m` (same for `wordpress-develop`); JSON reports via `go run ./cmd/syntax-metrics --root test_projects/symfony --json`.

### Practical CI target

- New `syntax` / trivia / binder packages: **≥95%** (or 100% if critical-unit-testing bar applies to those packages only).
- Whole-module line coverage: **non-goal** during cutover; round-trip + gold + analyse suite are the merge bar.

**Summary:** 100% on the identity/gold contract, very high on the new kernel, corpus/analyse green for integration — not “100% of go-php-parser.”
