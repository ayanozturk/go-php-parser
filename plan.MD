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
- [x] Eliminate the reproducible external-project false-positive batch recorded on 2026-09-22 with neutral parser/analyser regressions and a sibling-development PHP Strom re-scan.
- [ ] Reduce every failing file to a neutral fixture with a clean control.
- [ ] Classify each failure as lexing, grammar, recovery/progress, lowering, or diagnostic-only.
- [ ] Fix shared grammar/recovery logic rather than adding corpus-specific exceptions.
- [ ] Add an explicit token-progress assertion or guard for every recovered non-progress class.
- [ ] Re-run both full corpus gates after isolated failures pass.

Gate result: both corpora passed on 2026-09-20. No failing file was exposed, so the reduction, classification, fix, progress-guard, and post-fix rerun steps above remain conditional rather than being marked as work performed. Evidence: `docs/benchmarks/2026-09-20-syntax-corpus-baseline.md`.

The external-project false-positive batch was explicitly prioritized by the user on 2026-09-22. The production-equivalent sibling-development PHP Strom scan indexed vendor symbols, scanned 2,016 reportable files, and returned 18 intentionally deferred project annotations with none of the reported false-positive families remaining. The extension's committed parser pin and delivery validation are still part of step 6. Private project code and identifiers must not enter committed fixtures.

Commands:

```bash
SYNTAX_CORPUS_DIR="$PWD/test_projects/symfony" GOWORK=off go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
SYNTAX_CORPUS_DIR="$PWD/test_projects/wordpress-develop" GOWORK=off go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
```

### 3. Complete high-risk syntax coverage

- [x] Audit gold-tree coverage for attributes, DNF types, qualified/relative/fully-qualified names, mixed-case keywords, declaration trivia, heredoc/nowdoc, and interpolation.
- [x] Add focused gold/recovery coverage for PHP 8.4 property hooks, dynamic/braced members, first-class callables, casts, `yield`, enum cases, class-constant attributes/values, and Elvis ternaries.
- [x] Expand Zend `token_get_all` parity fixtures only for high-value lexer edge cases.
- [x] Add malformed and incomplete editor-input controls for each recovery change.
- [ ] Keep new `syntax`, trivia, lexer-state, and binder paths at least 95% covered, with branch-complete new state machines.

Step-3 evidence (2026-09-22 iteration, no grammar changes — characterization only): pre-existing gold covered attributes, DNF, names, heredoc/interpolation, and property hooks; added `syntax/gold_test.go` gold trees for Elvis (`TestGoldTreeElvisTernary`), all yield forms (`TestGoldTreeYieldForms`), dynamic/braced members (`TestGoldTreeDynamicBracedMembers`), first-class callables (`TestGoldTreeFirstClassCallables`), casts (`TestGoldTreeCasts`), attributed enum cases (`TestGoldTreeEnumCases`), attributed typed class consts (`TestGoldTreeClassConstAttributes`), nowdoc (`TestGoldTreeNowdoc`), mixed-case keywords (`TestGoldTreeMixedCaseKeywords`), and decl-trivia/PHPDoc attachment (`TestGoldTreeDeclTriviaAttachment`). Added `syntax/recovery_hotspots_test.go` (`TestHotspotIncompleteRecoveryControls`, 10 incomplete-input controls: identity + hotspot-kind survival + bounded diagnostics + `KindMissing` where the grammar emits one). Extended `TestTokenGetAllFullOrderedEquality` with 5 high-value parity fixtures (Elvis, first-class callable, enum, typed class const, dynamic members). Deliberately excluded `yield from` and `(int)`-style casts from parity: Zend emits combined `T_YIELD_FROM`/`T_INT_CAST` while this lexer emits split tokens by design (parser reassembles them). Gates: `GOWORK=off go test ./syntax ./lexer ./analyse -count=1` green; `go vet ./syntax ./lexer` clean; `git diff --check` clean.

Step-3 coverage slice (2026-09-23): added nil/wrong-kind controls for the condition, body, class/function, and call accessors in `syntax/conditions_test.go`, plus alternate `for` body-form coverage. `GOWORK=off go test ./syntax -coverprofile=/tmp/go-php-parser-syntax-coverage-after.out -count=1` passes; syntax package statement coverage moved from 82.3% to 82.8%, and `conditions.go:ForBody` moved from 81.8% to 90.9%. A paired pre-change `GOWORK=off go test ./syntax ./lexer -coverprofile=...` passed with lexer at 95.9%. `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`, and `git diff --check` pass. The syntax-wide 95% target remains open; this slice does not claim the whole new syntax/binder/trivia surface meets it.

Step-3 expression-accessor slice (2026-09-23): added nil and mismatched-kind controls for expression, literal, member, argument, unary, array, and label accessors in `syntax/coverage_accessors_test.go`. The syntax package passes with 83.2% statement coverage; `ParenInner`, `VariableName`, `ArrayElements`, `NewIsAnonymous`, and `LabelOrGotoName` now each reach 100%, and `ArgExpr` reaches 100%. The 95% new-kernel target remains open.

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
