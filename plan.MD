# Current implementation plan

Status: in progress

Roadmap source: promoted from the short-term roadmap

Scope: stabilize the lossless syntax cutover

This file is the single source of truth for work currently being implemented. Do not start a later `roadmap.md` item while this plan remains active unless the user explicitly changes priority. Update this file when a task is completed, blocked, split, or materially re-scoped.

## Objective

Turn the completed syntax-parser cutover into a release-quality baseline with complete corpus accounting, zero unexplained parser diagnostics, comprehensive lossless/gold/token coverage, safe recovery and cancellation, and a reproducible process-cold performance baseline.

## Work sequence

### 1. Capture the exact baseline

- [x] Record the parser commit, Go version, machine, corpus revisions, and clean/dirty worktree state.
- [x] Run focused `syntax`, `lexer`, and binder tests before corpus work.
- [x] Run `syntax-metrics` for Symfony and WordPress and retain machine-readable output outside the repository.
- [x] Record discovered, parsed, diagnostic-failure, byte, token, and identity counts for both corpora.

Baseline evidence: `docs/benchmarks/2026-09-20-syntax-corpus-baseline.md`.

Commands:

```bash
GOWORK=off go test ./syntax ./lexer ./analyse -count=1
GOWORK=off go run ./cmd/syntax-metrics --root "$PWD/test_projects/symfony" --json
GOWORK=off go run ./cmd/syntax-metrics --root "$PWD/test_projects/wordpress-develop" --json
```

### 2. Close identity, diagnostic, and recovery failures

- [x] Run the lossless identity gate on Symfony with an absolute corpus path.
- [x] Run the lossless identity gate on WordPress with an absolute corpus path.
- [ ] Reduce every failing file to a neutral fixture with a clean control.
- [ ] Classify each failure as lexing, grammar, recovery/progress, lowering, or diagnostic-only.
- [ ] Fix shared grammar/recovery logic rather than adding corpus-specific exceptions.
- [ ] Add an explicit token-progress assertion or guard for every recovered non-progress class.
- [ ] Re-run both full corpus gates after isolated failures pass.

Gate result: both corpora passed on 2026-09-20. No failing file was exposed, so the reduction, classification, fix, progress-guard, and post-fix rerun steps above remain conditional rather than being marked as work performed. Evidence: `docs/benchmarks/2026-09-20-syntax-corpus-baseline.md`.

Commands:

```bash
SYNTAX_CORPUS_DIR="$PWD/test_projects/symfony" GOWORK=off go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
SYNTAX_CORPUS_DIR="$PWD/test_projects/wordpress-develop" GOWORK=off go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
```

### 3. Complete high-risk syntax coverage

- [ ] Audit gold-tree coverage for attributes, DNF types, qualified/relative/fully-qualified names, mixed-case keywords, declaration trivia, heredoc/nowdoc, and interpolation.
- [ ] Add focused gold/recovery coverage for PHP 8.4 property hooks, dynamic/braced members, first-class callables, casts, `yield`, enum cases, class-constant attributes/values, and Elvis ternaries.
- [ ] Expand Zend `token_get_all` parity fixtures only for high-value lexer edge cases.
- [ ] Add malformed and incomplete editor-input controls for each recovery change.
- [ ] Keep new `syntax`, trivia, lexer-state, and binder paths at least 95% covered, with branch-complete new state machines.

### 4. Validate cancellation and robustness

- [ ] Run parser security and malformed-input fuzz seeds.
- [ ] Run focused race tests for parser, binder, and corpus paths.
- [ ] Verify cancellation interrupts pathological and large-file parses without leaving partial shared state.
- [ ] Verify recovery produces bounded diagnostics and preserves source identity.

### 5. Establish an accepted parser baseline

- [ ] Measure `ParseCST`, `LowerAST`, and `ParseAndLowerAST` separately.
- [ ] Profile only after file and diagnostic accounting matches the correctness baseline.
- [ ] Run process-cold measurements in isolation and reject any result with CV above 5%.
- [ ] Record time, allocations, peak RSS, files, bytes/tokens, diagnostics, corpus revisions, parser commit, and machine details.
- [ ] Compare with the exact previous parser binary without changing the workload.
- [ ] Write the accepted result to `docs/benchmarks/`; keep raw JSON and profiles outside the repository.

### 6. Integration and delivery

- [ ] Run `git diff --check`, `GOWORK=off go test ./...`, and `GOWORK=off go vet ./...`.
- [ ] Run applicable race, fuzz-seed, identity, gold, token-parity, and corpus gates.
- [ ] Push the parser commit only after the intended diff and validation are complete.
- [ ] Update PHP Strom's `server/go.mod` pin.
- [ ] Validate PHP Strom in pinned-module and sibling-development modes.
- [ ] Verify local and remote parser refs and the committed PHP Strom pin.
- [ ] Mark this plan complete, archive its durable evidence, and promote the next roadmap item into a fresh `plan.MD` only after completion is verified.

## Non-negotiable acceptance gates

- `Print(Parse(src)) == src` for every fixture and corpus file that lexes, including malformed-input recovery through missing/error tokens.
- Every discovered file is accounted for; every remaining parser diagnostic is classified and documented.
- No private corpus source, paths, messages, or identifiers enter committed fixtures.
- Performance comparisons use identical workloads and pass the 5% CV limit.
- Diagnostic totals are not treated as semantic parity evidence.
- No parser version is described as delivered through PHP Strom until the engine commit is pushed and both integration modes pass.

## Out of scope for this plan

- Porting the remaining AST-only analysis rules to CST.
- Expanding PHPStan diagnostic families or framework metadata unless required to classify a parser regression.
- CLI feature work, formatter work, editor latency optimization, or broad semantic-system additions.
- Allocation or GC work not justified by the accepted parser profile.
