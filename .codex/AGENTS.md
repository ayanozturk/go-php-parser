# Subagent delegation

Proactively delegate bounded, low-risk work to the `worker` subagent when it can complete the task independently and doing so keeps the main agent focused or reduces expensive main-agent usage.

Prefer the `worker` for:

- small, clearly scoped implementation tasks
- repetitive or mechanical edits
- focused test additions or updates
- narrow bug fixes with an obvious validation path
- straightforward refactors that follow existing project patterns
- repository searches, call-site discovery, and other bounded investigation supporting an implementation task

Prefer delegation over doing routine implementation in the main agent when the worker can complete it independently.

Keep with the main agent:

- architecture and design decisions
- ambiguous or open-ended investigations
- complex debugging where the root cause is unclear
- cross-cutting changes requiring broad coordination
- release, deployment, or migration work
- security-sensitive or high-risk changes
- final integration and review

Do not run multiple write-capable agents on overlapping files or overlapping areas of responsibility.

When delegating:

- give the worker a narrow objective and clear acceptance criteria
- identify relevant files or constraints when known
- avoid delegating unnecessary surrounding context
- keep ownership of architectural decisions with the main agent

The main agent remains responsible for:

- reviewing all worker changes
- resolving integration issues
- checking that scope was not widened unnecessarily
- running appropriate project-level validation
- confirming the final implementation is coherent before completion

# Commit and push authorization

The user has explicitly authorized committing and pushing completed, in-scope changes for this project.

This authorization does not broaden task scope.

Before committing or pushing:

- review the final diff
- run relevant tests, linting, type checks, or other appropriate validation
- avoid including unrelated changes
- use a concise commit message that accurately describes the completed work

Do not commit or push incomplete, unvalidated, or out-of-scope changes.

# Testing coverage (lossless parser cutover)

Do **not** chase a single line-% on the whole repo. Use layered targets.

## Must-hold (correctness gates)

- **100% lossless invariant** on fixtures + corpus: `Print(Parse*(src)) == src` for every file that lexes (parse-error cases round-trip via missing/error tokens).
- **Gold trees** for lossy hotspots: attributes, DNF types, names (qualified/relative/FQ), heredoc/nowdoc + interpolation, mixed-case keywords, trivia attachment on decls.
- **Token parity** fixtures vs `token_get_all` (kind + text) for Zend edge cases — small set, high value.

## Package-level (unit)

- `syntax` + new binder paths: aim very high (≈90–100% of *new* code).
- Lexer state machines (attribute/`#[`, encapsed, heredoc): branch-complete on new states, not “lines in lexer.go”.
- Legacy `ast` / old string parsers: coverage may fall as code is deleted — don’t maintain 100% on code being removed.

## Analyse / Strom

- Prefer **behavioral coverage**: binding, rename/refs, format identity, type-from-syntax — not line % on giant rule files.
- Keep differential/level fixtures green; add cases when string→syntax edges move.

## Perf (not coverage %)

- Bench gates: allocs/op, tokens/KB on Symfony/WordPress-sized inputs — regressions fail CI even if coverage is high.

## Practical CI target

- New `syntax` / trivia / binder packages: **≥95%** (or 100% if critical-unit-testing bar applies to those packages only).
- Whole-module line coverage: **non-goal** during cutover; round-trip + gold + analyse suite are the merge bar.

**Summary:** 100% on the identity/gold contract, very high on the new kernel, corpus/analyse green for integration — not “100% of go-php-parser.”
