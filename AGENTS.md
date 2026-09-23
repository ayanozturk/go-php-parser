# Codex Instructions

This repo is the Go PHP parser and static analyser. Sibling LSP delivery lives in `vscode-php-strom` (`server/go.mod` pins this parser).

## Planning files

- `plan.MD` is the single source of truth for the implementation currently in progress. Read it before starting repository work and keep changes within its stated scope.
- `roadmap.md` contains future short-, medium-, and long-term work. Do not start or promote a roadmap item while `plan.MD` is active unless the user explicitly changes priority or the current plan is verified complete.
- Capability inventories, benchmark reports, rule pages, and historical progress documents are evidence and reference material, not alternative queues.
- Update `plan.MD` when an active task is completed, blocked, split, or materially re-scoped. Check off work only after its acceptance evidence exists.
- When the current plan is complete, verify delivery first, then replace its contents with the next approved roadmap item. Do not accumulate completed-plan history in `plan.MD` or completed-work history in `roadmap.md`.

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

### CST-direct analysis (architecture reference, not a progress log)

`analyse/` rules walk `*syntax.RedNode` directly (`syntax.Walk`,
`syntax.RedNode.Pos()`/`EndPos()`) instead of lowering the whole file to
`[]ast.Node` first. `analyse/syntax_walk.go`'s `walkSyntaxConfigured` is the
shared traversal driver (class/function scope tracking, a small "pure
container kind" exclusion set) that most context-dependent rules use;
`analyse/syntax_file_type_context.go`'s `CollectFileTypeContextFromSyntax`
builds the CST-direct `FileTypeContext`/resolver scope. `syntax/conditions.go`
provides reusable condition/body accessors (`IfCondition`, `WhileCondition`,
`ForConditions`, `MatchCondition`, `FunctionBody`, `ClassMethods`, etc.) so
rules don't re-derive child-detection logic from the lowering code.

Many rules still lower a *matched CST subtree* back to `ast.Node` via
`syntax.LowerStmtNode`/`LowerExprNode`/`LowerPropertyDeclNode`/`LowerParamNode`/
etc. and call the existing, unmodified `*OnNode`/`append*OnNode` check
functions (`checkLanguageOnNode`, `checkTypeReferenceOnNode`,
`checkSymbolOnNode`, `appendClassModelOnNode`, `appendReturnTypeOnNode`, ...)
as shared leaf logic — these are not dead code and must not be deleted even
though the file-wide `[]ast.Node` walk they used to run under is gone.

`AnalysisContext.Content []byte` (`context.go`) selects the CST-direct
production paths in `RunAnalysisRulesWithContext`. The registry runner
reuses the caller's `*syntax.ParseResult`; the fused rules and shared
call/assignment and unreachable-code passes consume that parse or cached
per-file facts. A small set of shared support facts (such as PHPDoc aliases
and reflection guards) is still extracted from the already-lowered AST.
Production callers, including `server/providers/diagnostics.go` in
`vscode-php-strom`, set `Content` and reuse the parse result. Callers that
omit it retain the AST-based compatibility entry points.

Known, intentionally-preserved gaps and gotchas when touching this area:

- A flat `syntax.Walk` from the root does **not** preserve the
  statement-vs-expression traversal boundary that hand-written `ast.Node`
  type switches got for free — rules that need it use a bespoke recursive
  statement walker with a separate, narrowly-scoped expression walker for
  isolated subtrees (see `analyse/assignment_in_condition_rule.go`).
- `walkSyntaxConfigured` deliberately mirrors several *pre-existing*
  ast.Node-side lowering/dispatcher blind spots for parity (not bugs to fix
  here): switch statement bodies, declarations nested inside a statement
  body, `yield`, enum cases, class-constant attributes/values,
  first-class-callable syntax, type casts, PHP 8.4 property hooks, and the
  Elvis (`?:`) ternary's shared condition/`IfTrue` node. Don't "fix" these
  ad hoc per rule — any real fix belongs in the shared walker or the
  `syntax/lower_*.go` pipeline, validated across both corpora below.
- `cmd/analysis-corpus-snapshot` is the safety net for any change here: run
  it (or an equivalent one-off diff) against `test_projects/symfony` and a
  composer-src checkout before/after to confirm 0 issue-set mismatches.

Remaining, acknowledged debt (deferred, not blocking):

- `AssignmentInConditionRule` and `SideEffectsRule` still keep their
  original `ast.Node`-based `CheckIssues` implementations alongside their
  CST-direct versions. Their AST paths remain compatibility APIs for callers
  without source content.
- Level 1 variable-flow facts can still be built lazily from AST nodes when a
  caller supplies `Content` without a shared `VariableFlow`; production
  `SemanticSnapshot` contexts provide the shared flow facts.

### Perf (not coverage %)

- Bench gates: allocs/op, tokens/KB on Symfony/WordPress-sized inputs — regressions fail CI even if coverage is high.
- Corpus identity + metrics: `SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m` (same for `wordpress-develop`); JSON reports via `go run ./cmd/syntax-metrics --root test_projects/symfony --json`.

### Practical CI target

- New `syntax` / trivia / binder packages: **≥95%** (or 100% if critical-unit-testing bar applies to those packages only).
- Whole-module line coverage: **non-goal** during cutover; round-trip + gold + analyse suite are the merge bar.

**Summary:** 100% on the identity/gold contract, very high on the new kernel, corpus/analyse green for integration — not “100% of go-php-parser.”
