# Default VS Code PHP Platform Progress

Last updated: 2026-09-01 (Europe/London)

This file records reproducible evidence for the cooperating `go-php-parser` engine and `vscode-php-strom` extension. PHPStan, PHPCS, and Mago are benchmark references only; no parity claim is made.

## Current baseline

- Engine revision validated and consumed by PHP Strom: pushed commit `41dfb55`.
- Extension checkout: `/Users/ayan/Projects/vscode-php-strom`, `main` at pushed commit `fa2ce9a`; the current package version is `0.1.34`.
- Production extension builds pin `github.com/ayanozturk/go-php-parser` at pseudo-version `v0.0.0-20260901192221-41dfb5580c97`. `make test-server-dev` validates the same engine through the generated, ignored sibling-workspace path.
- Go toolchain observed: Go 1.27.0. Node toolchain observed: Node 26.8.1 and npm 11.19.0.
- Representative corpora are fetched at exact revisions from `test_projects/manifest.json`; generated working copies remain uncommitted.
- The latest recorded full-corpus pass has zero failures for Composer, Drupal, Magento, PHPUnit, and WordPress. Symfony's two remaining fixtures are intentionally invalid/corrupted inputs, and Laravel has two narrow interpolation/callable edge cases. These recorded results are compatibility evidence, not a current performance result.

## Security and fuzzing

- `npm audit --audit-level=low`: 0 known vulnerabilities across 531 installed packages on 2026-09-01. Extension commit `59eb0b0` refreshes `browserslist` from 4.28.2 to 4.28.8 and updates its lockfile-only metadata dependencies.
- GitHub remote verification after push: 0 open Dependabot alerts, 33 fixed Dependabot alerts, and 0 open secret-scanning alerts. The push-time banner briefly reported 26 alerts while GitHub recalculated the default branch; the alerts API is the recorded final state.
- `govulncheck`, `gosec`, `staticcheck`, and `golangci-lint` were not installed; no result is claimed for them.
- Both Go repositories now have deterministic malformed-PHP fuzz targets. Pinned GitHub Actions smoke jobs run the parser panic/cancellation targets for 15s/10s and the production index/diagnostic wrapper for 20s, each with a 10-minute job timeout and read-only repository permissions.
- The parser recovers panics into `Parser panic:` errors and supports cooperative context cancellation. The language-server indexer applies a 20-second per-file parse timeout and a four-worker cap.
- Before the 2026-08-19 change, the indexer used `os.ReadFile` before enforcing `files.maxSize`, so the configured gate did not bound allocation for oversized or growing files.
- Save, close, and workspace-diagnostic disk reloads now apply `files.maxSize`, reject non-file or non-regular resources, resolve symlinks before validating configured workspace roots, and retain bounded single-file mode when no workspace is open.
- Workspace discovery now rejects symlinked and special directory entries before association matching. On macOS/Linux, bounded reads additionally use nonblocking, no-follow opens and reject non-regular descriptors, closing the static symlink escape, named-pipe hang, and discovery-to-open substitution paths without a per-file metadata lookup during discovery.
- The Windows/other-platform bounded-reader fallback rejects non-regular descriptors after opening; Windows workspace discovery rejects reparse-point symlinks and filesystem special entries from `DirEntry.Type`. The nonblocking/no-follow open hardening is compiled for the supported macOS/Linux targets where POSIX named pipes are reachable directory entries.

## Performance baseline

Command:

```sh
cd /Users/ayan/Projects/vscode-php-strom/server
GOWORK=off go run ./cmd/benchmark-indexer /Users/ayan/Projects/go-php-parser/test_projects
```

Representative workload: 23,556 indexed PHP files, 3,678,678 LOC, 135.68 MB, 151,253 symbols, four workers, function bodies skipped, Composer/vendor paths excluded by the benchmark config.

- First pre-change run: 1.963s, 12,001 files/s, 1,874,230 LOC/s, 620.92 MB HeapAlloc, 735.10 MB Sys.
- An initial bounded-reader implementation regressed to 2.279s and was not retained.
- Final implementation warm runs: 1.799s, 1.727s, 1.685s (median 1.727s); median HeapAlloc 613.25 MB and median Sys 755.48 MB.
- Cache warmth differs between the single first run and the three final runs. These measurements show the initial regression was removed, but they do not establish a performance improvement.
- No equivalent local PHPStan or PHPCS executable was available. Mago was available at `/opt/homebrew/bin/mago`, but no equivalent configured analysis workload was established, so no cross-tool timing claim is recorded.
- 2026-08-21 interleaved A/B runs compared the changed extension with a detached `23dc399` checkout on the same 23,556-file, 135.68 MB workload. All ten measured runs indexed 23,556 files and 151,587 symbols. Changed times were 1.974s, 1.924s, 1.875s, 1.942s, and 1.891s (median 1.924s); baseline times were 1.949s, 1.885s, 1.917s, 1.842s, and 1.936s (median 1.917s). The +0.4% median difference is within run noise; no performance improvement or regression is claimed.
- 2026-08-22 sequential parser microbenchmarks (`go test ./parser -run='^$' -bench='^BenchmarkParse$' -benchtime=500ms -count=10`) compared detached `4350d06` with the final nil-guard placement. Baseline median was 8,769.5 ns/op and changed median was 8,848.5 ns/op (+0.9%); ranges overlap and the runs were not interleaved, so no performance regression or improvement is claimed. The retained guard executes only after a fallible postfix parse, not on ordinary expressions.

## Maintainability and architecture risks

- `FEATURES.md` still contains leftover TypeScript/tree-sitter feature prose, but the architecture overview now states that production is the Go language server and `go-php-parser`. A full specification rewrite remains a ranked maintenance item.
- The obsolete `src/server` TypeScript implementation is excluded from both the production TypeScript build and ESLint; retaining dead server code remains an architecture/deletion decision requiring separate evidence.
- Parser AST nodes and structured analysis issues expose complete spans, and PHP Strom now converts their one-based rune coordinates against the exact source text into half-open UTF-16 LSP ranges. Parser errors still expose an unstructured string API, and style-rule coordinates come from mixed legacy producers, so those diagnostics retain point ranges pending a separate coordinate-contract migration.

## Coverage gaps

- Differential benchmark: level 0 remains partial but is gated by 88 reviewed fixtures. Level 1 is gated by 24 variable-flow fixtures; level 2 by 65 protected/unknown-method/non-object/dynamic-index/expression-receiver fixtures; level 3 by one throw-type fixture; level 7 by five partial-union fixtures; level 8 by five nullable-object fixtures. Pinned reference: `PHPStan 2.2.x-dev@e4ab62a`. Remaining analyser gaps: remaining silent or extra-identifier level-0 leftovers, the reference analyser’s level-8 reclassification of unknown nullable methods, narrowing, generic inheritance, higher-level return/property/argument checks, PHPDoc validation, dynamic-call precision, and extension-dependent built-in signatures.
- PHPCS benchmark: the repository comparison records 16 style rules, far below the breadth of PHPCS standards. Security-oriented source rules such as eval/backtick/forbidden-function checks are not implemented.
- The remaining recorded pure-parser corpus gaps are two narrow Laravel vendor-code cases; intentionally invalid or corrupted fixtures stay classified separately from parser failures.
- PHP Strom now consumes shared `SemanticSnapshot` facts, flow graphs, and variable-flow state through dependency-scoped exported-semantic revisions, changed documents use immutable incremental project-index replacement, structured analysis diagnostics use UTF-16 spans, and a synthetic editor-path trace suite accounts for cache, dependency, cancellation, publication, and fallback behavior. Remaining extension integration gaps include stale architecture documentation, unstructured parser errors and mixed style-rule point ranges, conservative name-based dependency false positives/global overflow fallback, and no full VS Code extension-activation or representative-project latency trace.

## Completed changes and validation

### 2026-08-19 — Security: bound workspace-index file reads

- Changed `vscode-php-strom/server/indexer/indexer.go` to reject regular files whose stat size exceeds `files.maxSize` before allocating their contents.
- Added a limit+1 reader fallback for files that grow after stat or do not report a reliable size; oversized content is discarded.
- Preserved normal behavior for files exactly at the configured limit.
- Added regression tests in `vscode-php-strom/server/indexer/indexer_test.go` proving an oversized reader consumes only limit+1 bytes, an oversized regular file is rejected, and a file exactly at the limit is retained.
- `GOWORK=off go test ./indexer -count=1`: pass.
- `GOWORK=off go test -race ./indexer -count=1`: pass.
- `GOWORK=off go test ./...`: pass.
- `GOWORK=off go vet ./...`: pass.
- `make test-server-dev`: pass against the sibling engine checkout.
- `go test ./...`, `go vet ./...`, and `go test -race ./...` in `go-php-parser`: pass.
- `npm run compile`: pass.
- `npm run package`: pass with the known `vscode-languageserver-types` dynamic-require warning.
- Six local-parser cross-builds (`darwin/linux/windows` x `arm64/amd64`) to a temporary directory: pass.
- Committed and pushed to `vscode-php-strom/main` as `aa6c7c7` after the dependency remediation commit `9dfa9c3`.

### 2026-08-20 — Maintenance: restore complete validation checks

- Added an ESLint 9 flat configuration using the official TypeScript recommended rules and aligned its scope with the production `tsconfig.json` inputs.
- Added a compiled VS Code 1.89.1 extension-host smoke test that verifies the development extension, core commands, and PHP language contribution are discoverable. Generated `.vscode-test` state is ignored.
- Replaced asynchronous test output buffers with a synchronized writer, removing the `TestDidChangeDebouncesOnTypeAnalysis` race without weakening the production server or race detector.
- `npm ci`: pass; 530 packages installed and 531 audited.
- `npm audit --audit-level=low`: pass; 0 vulnerabilities.
- `npm run lint`: pass.
- `npm run compile`: pass.
- `npm run package`: pass with the existing `vscode-languageserver-types` dynamic-require warning.
- `npm test`: pass in the pinned VS Code 1.89.1 extension host.
- `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`, and `GOWORK=off go test -race ./...`: pass.
- `make test-server-dev`: pass against the sibling engine checkout.
- Engine `go test ./...`, `go vet ./...`, and `go test -race ./...`: pass.
- Committed and pushed to `vscode-php-strom/main` as `7889e04`. The audited lockfile remediation was committed separately as `9dfa9c3`.

### 2026-08-20 — Features: parse division assignment and locate syntax diagnostics

- Reproduced a false-positive cascade in `/Users/ayan/Projects/hr/src/Command/GenerateApiDocsCommand.php`: valid `$bytes /= 1024;` was lexed as separate `/` and `=` tokens, producing nine editor diagnostics. `php -l` and `mago analyse` both accepted the file.
- Added lexer coverage proving `/` remains `T_DIVIDE` while `/=` emits the existing `T_DIV_EQUAL` token, plus a parser regression test for division assignment in a function body.
- Updated `vscode-php-strom` parse diagnostics to derive LSP ranges from the parser's `line N:C:` prefix instead of highlighting line 1 for every syntax error; unstructured panic/recovery messages retain the safe line-1 fallback.
- Local-parser `cmd/parse-test` scan of the reported HR file: 1 file, 0 timeouts, 0 panics, 0 parse errors.
- Engine `go test ./...`, `go vet ./...`, and `go test -race ./...`: pass.
- Extension `make test-server-dev`: pass against the sibling engine checkout, including the diagnostic-range tests.
- Extension pinned-module `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`, and `GOWORK=off go test -race ./...`: pass.
- Extension `npm run lint`, `npm run compile`, and `npm test`: pass; `npm audit --audit-level=low` reports 0 vulnerabilities.
- Extension `npm run package`: pass with the existing `vscode-languageserver-types` dynamic-require warning.
- No benchmark was run because this is a constant-time lexer branch and diagnostic-location correctness fix, not a performance optimization; no performance improvement is claimed.
- Committed and pushed the engine fix as `1bdc065`; committed and pushed the pinned dependency and diagnostic-location integration as `f3d205b`.

### 2026-08-20 — Security: bound and confine diagnostic disk reloads

- Reproduced three reachable failures in `vscode-php-strom/server/phpstrom/handler_save_test.go`: `didSave` replaced bounded in-memory text with oversized disk content, while `didClose` re-read and indexed both an outside-workspace file and an in-workspace symlink targeting a file outside the workspace.
- Reused the indexer's limit+1 reader for save, close, and workspace-diagnostic disk reads, enforcing the current `phpstrom.files.maxSize` without allocating oversized contents.
- File URI handling now rejects non-file schemes, opaque/query/fragment forms, relative paths, and unsupported authorities; it preserves encoded in-workspace paths, Windows drive paths, Windows UNC authorities, and bounded single-file mode.
- Disk reloads resolve symlinks and regular-file metadata before checking canonical target paths against canonical workspace roots. Unknown `didClose` notifications cannot initiate disk reads.
- Adversarial tests cover oversized reloads, encoded valid paths, non-file/relative/opaque/query/fragment URI rejection, outside-workspace paths, symlink escapes, and forged `didClose` notifications for unopened documents.
- Targeted `GOWORK=off go test ./phpstrom ./indexer -count=1`, `GOWORK=off go test -race ./phpstrom ./indexer -count=1`, and scoped `go vet`: pass.
- Extension `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`, and `GOWORK=off go test -race ./...`: pass.
- `make test-server-dev`: pass against sibling engine commit `1bdc065`; six pinned-parser builds (`darwin/linux/windows` x `arm64/amd64`): pass.
- Clean `npm ci`: pass; 530 packages installed and 531 audited. `npm audit --audit-level=low`: pass with 0 vulnerabilities.
- `npm run lint`, `npm run compile`, `npm run package`, and `npm test`: pass; packaging retains the documented `vscode-languageserver-types` dynamic-require warning and the extension-host smoke test passed in VS Code 1.89.1.
- Engine `go test ./...`, `go vet ./...`, and `go test -race ./...`: pass.
- A first implementation added a metadata syscall to every indexed file and produced warm runs of 2.122s, 1.960s, and 1.939s (median 1.960s versus the recorded 1.727s median); it was not retained.
- The final index-reader hot path differs only by exported symbol name: baseline and changed functions are both 720 bytes and have identical opcode hashes (`eaeb3035c153cb2301335880b3c19b0e82fc0cf4db0a0698df817f62aacefdb3`). Interleaved wall-clock runs were noisy under concurrent load, so no timing improvement or parity claim is made.
- Committed the extension change locally as `a6ab54d`; no push, release, publication, or marketplace installation was performed.

### 2026-08-20 — Features: accept `for` as a contextual method name

- Reproduced valid PHP using `public static function for(...)` and `TaskStatusDisplay::for(TaskStatus::BACKLOG)`; PHP accepts the enum case and contextual keyword method name, while the parser rejected `T_FOR` in both positions.
- Added an end-to-end parser regression covering a backed enum, the keyword-named declaration, and the static call with an enum-case argument.
- The HR `src/ValueObject` corpus improved from 1 parse-error file to 0 across 81 files; the full `src` corpus improved from 14 to 13 parse-error files across 1,091 files, with 0 panics and 0 timeouts.
- `go test ./...`, `go vet ./...`, and `git diff --check`: pass. Extension server `go test ./...`, TypeScript compile, webpack package, and VSIX packaging: pass; webpack retains its known dynamic-require warning.
- This is a constant-time token classification correction; no performance improvement is claimed.

### 2026-08-20 — Features: eliminate remaining HR parser false positives

- Verified all 13 remaining parser-failing HR files with PHP 8.4 `php -l`; every file was valid PHP.
- Grouped the failures into four shared grammar gaps: `new` as a contextual method name, trailing comments before a class closing brace, `mixed` typed properties, and comments between a property type and variable.
- Added adversarial regression coverage combining the four forms, and extended the corpus scanner to print at most five diagnostics per failing file for reproducible triage.
- The full `/Users/ayan/Projects/hr/src` corpus improved from 13 parser-failing files to 0 across 1,091 files, with 0 panics and 0 timeouts.
- Engine `go test ./...`, `go vet ./...`, `go test -race ./...`, and `git diff --check`: pass.
- Extension local-parser and pinned-parser tests pass, including race and vet; `npm run lint`, `npm run compile`, `npm run package`, `npm audit --audit-level=low`, and `git diff --check`: pass. Audit reports 0 vulnerabilities; packaging retains the known dynamic-require warning.
- These changes add constant-time token checks and skip already-tokenized comments; no performance improvement is claimed.

### 2026-08-20 — Features: validate and clear full-body HR diagnostics

- Found that `cmd/parse-test` used the symbol-indexing path, which intentionally skipped function bodies and could not reproduce editor diagnostics in `Mail/OnboardingEmail.php`; changed it to the full production diagnostic parser and retained bounded error details.
- Reproduced and fixed direct fluent calls on constructor expressions such as `new TemplatedEmail()->from(...)`, with a focused regression test.
- The corrected full-body scan exposed and fixed one final false positive for keyword-named arguments such as Symfony Regex's `match: false`, also with regression coverage.
- PHP 8.4 `php -l` and the full diagnostic parser now both accept `Mail/OnboardingEmail.php` and `Form/ContactUsFormType.php`.
- Corrected full-body scan: 1,091 HR source files, 0 parser-error files, 0 panics, and 0 timeouts.
- Engine full tests, vet, race tests, and diff checks pass. Extension local-parser tests, lint, compile, package, extension-host tests, audit, and diff checks pass; audit reports 0 vulnerabilities and packaging retains the known dynamic-require warning.
- The parser changes reuse the existing postfix loop and add a constant-time identifier classification; no performance improvement is claimed.

### 2026-08-20 — Maintenance: group syntax diagnostics as Parser Errors

- Parser diagnostics previously had no LSP code, causing the project Problems tree to create one group per raw recovery message.
- Assigned every parser diagnostic the stable `Parser Errors` code; the existing view now groups genuine syntax findings together while analysis and style findings retain their rule-specific groups.
- Added a provider regression proving syntax diagnostics retain the grouping code with parser debug disabled.
- Pinned and local-parser provider tests pass. Extension full Go tests, vet, race tests, lint, compile, package, extension-host tests, audit, and diff checks pass; audit reports 0 vulnerabilities and packaging retains the known dynamic-require warning.
- This changes diagnostic metadata and view organization only; parsing, severity, ranges, messages, security boundaries, and performance are unchanged.

### 2026-08-20 — Features: parse HR anonymous classes and flexible heredocs

- Verified `tests/Unit/Service/FeatureCheckerRefactoredTest.php` and `migrations/Version20250704150715.php` with PHP 8.4 `php -l`; both parser findings were false positives.
- Anonymous classes now accept a trailing comment before the closing brace and share normal-class handling for `mixed` properties and comments between a type and property variable.
- Heredoc/nowdoc lexing now accepts PHP's indented closing marker, permits a closing identifier followed by `)`, and removes the closing indentation from body lines. Regression tests verify token continuation and the dedented value.
- Empty and whitespace-only PHP files are accepted like `php -l`, while non-empty files without `<?php` retain the missing-open-tag diagnostic.
- Full-body scans: 771 HR unit-test files and 79 migration files, 0 parser errors, 0 panics, and 0 timeouts.
- Engine full tests, vet, race tests, and diff checks pass. Extension local-parser tests, lint, compile, package, extension-host tests, audit, and diff checks pass; audit reports 0 vulnerabilities and packaging retains the known dynamic-require warning.
- The lexer performs a linear per-line indentation check already bounded by source size; no performance improvement is claimed.

### 2026-08-20 — Features: run full workspace analysis after VS Code reload

- Changed `phpstrom.diagnostics.workspaceScanOnStart` from opt-in to enabled by default, so fresh activation and a full VS Code reload index and analyse every associated workspace PHP file through the existing bounded workspace scan.
- Preserved an explicit `false` user/workspace setting as an opt-out for unusually large projects; manual `PHP Strom: Refresh Problems Scan` remains available.
- Added an extension-host manifest regression proving the shipped default is enabled and aligned the runtime fallback with the manifest.
- Pinned/local server tests, vet, race tests, TypeScript lint/compile/package, extension-host tests, audit, and diff checks pass. Audit reports 0 vulnerabilities; packaging retains the known dynamic-require warning.
- Startup now intentionally performs the configured full scan, so reload cost scales with the associated, non-excluded workspace PHP corpus and existing file-size/diagnostic caps; no startup-speed improvement is claimed.
- Release validation completed for extension 0.1.19 (patch from 0.1.18): engine full tests/vet/race, pinned and local-parser server tests/vet/race, TypeScript lint/compile/package, extension-host tests, npm audit with 0 vulnerabilities, manifest-lockfile consistency, and both repositories' diff checks passed.

### 2026-08-21 — Security: reject symlinked and special workspace entries

- Reproduced the boundary failure with an adversarial workspace containing one regular PHP file, an outside-workspace PHP symlink, a symlinked directory, and a PHP-named FIFO: discovery incorrectly returned the regular file, symlink, and FIFO.
- Discovery now checks the `os.DirEntry` type before association matching, skipping symlinks and non-regular entries without calling `Info` for ordinary files. macOS/Linux bounded reads use one nonblocking, no-follow open plus the existing descriptor stat; symlink and FIFO reads are rejected, and FIFO regression coverage proves the call returns within one second.
- Corrected a pre-existing extension-host baseline failure found during validation: commit `a8088a7` changed the runtime fallback and smoke-test contract for `phpstrom.diagnostics.workspaceScanOnStart` to `false`, but the shipped manifest remained `true`. The manifest now matches the newer performance decision, so startup still indexes symbols while full workspace diagnostics remain available manually or via explicit opt-in.
- Focused `GOWORK=off go test ./indexer` and `GOWORK=off go test -race ./indexer` adversarial runs: pass. Indexer test binaries compile for `darwin/linux/windows` on `arm64/amd64`: pass.
- Extension pinned-module `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`, and `GOWORK=off go test -race ./...`: pass. `make test-server-dev` plus sibling-parser Go vet/race: pass. `make build-server` builds all six marketplace targets.
- Engine `go test ./...`, `go vet ./...`, and `go test -race ./...`: pass.
- Clean `npm ci`: pass (530 packages installed, 531 audited). `npm audit --audit-level=low`: pass with 0 vulnerabilities. `npm run lint`, `npm run compile`, `npm run package`, and the VS Code 1.89.1 extension-host `npm test`: pass; webpack retains the known `vscode-languageserver-types` dynamic-require warning.
- Version-sensitive checks after the patch bump from 0.1.22 to 0.1.23 confirmed manifest/lockfile consistency and repeated lint, compile, webpack package, extension-host tests, audit, and `git diff --check` successfully.
- `make install`: pass; rebuilt all six server targets, packaged `phpstrom-0.1.23.vsix` (24 files, 19.89 MB), and installed it successfully into local VS Code. Packaging retains the known dynamic-require warning; the VS Code CLI also emitted its existing Node `url.parse()` deprecation warning.
- A malformed zsh cross-build loop produced no valid build result and is not counted. The corrected six-target `go test -c` builds and later `make build-server` both pass. A sibling-parser vet command initially ran from the repository root and failed module discovery; the same command passed from `server/` with the generated Go workspace.

### 2026-08-22 — Security: fuzz malformed PHP and stop nil postfix recovery

- Baseline `go test ./parser -list '^Fuzz'` and pinned extension `go test ./indexer -list '^Fuzz'` listed no fuzz targets, leaving malformed/untrusted input outside continuous adversarial coverage.
- Added bounded 64 KiB fuzz targets for the engine parser and the extension's production full-diagnostic and symbol-index parse paths. Deterministic seeds include truncated structures, unterminated heredoc/comments, invalid UTF-8, and a minimized chained-postfix reproducer. The targets fail on any recovered `Parser panic:` diagnostic and prove pre-cancelled parses report cancellation.
- The first five-second engine run found a real nil-pointer panic from `<?phpA[0(00`: a failed array-access parse returned `nil`, but the postfix loop continued into a following call expression and dereferenced it. The loop now returns immediately after any fallible postfix parser fails, covering object, static, array, and variable-call chains.
- Added read-only, SHA-pinned GitHub Actions workflows in both repositories. The engine budgets malformed/cancellation fuzzing at 15s/10s; the extension budgets its production-wrapper target at 20s. Each job has a 10-minute outer timeout.
- Final local fuzz evidence: malformed parser target 1,886,552 executions in 10s; cancellation target 2,010,941 executions in 10s; production indexer target 1,130,906 executions in 10s. Deterministic seed runs pass in both repositories.
- Engine `go test ./...`, `go vet ./...`, `go test -race ./...`, and `git diff --check`: pass. The five-project compatibility scan completed without a process crash in 2.716s with 31,686/32,990 files compatible, 1,304 failing files, and 146,218 parse errors.
- Extension `make test-server-dev`, sibling-parser Go vet/race, `npm run lint`, `npm run compile`, `npm run package`, VS Code 1.89.1 extension-host `npm test`, clean `npm ci`, `npm audit --audit-level=low`, and `git diff --check`: pass. Audit reports 0 vulnerabilities; packaging retains the known `vscode-languageserver-types` dynamic-require warning.
- This security fix adds nil checks only after fallible postfix parsing. The sequential microbenchmark median was +0.9% with overlapping ranges; no performance change is claimed.

### 2026-08-29 — Integration: reuse parser semantic snapshots in PHP Strom

- Extension commit `15abf7a` constructs parser-native `SemanticSnapshot` instances over the current document and immutable workspace project index, then supplies their shared semantic facts, control-flow graph, and variable-flow reader to diagnostics, hover, definition, and declaration analysis.
- At that checkpoint cached semantic snapshots were keyed by exact document text and an explicit workspace project revision. Unchanged requests reused the same immutable readers; any document edit or cross-file project rebuild invalidated them. The 2026-08-30 batch below narrows cross-file invalidation to exported changes.
- Closed/background workspace diagnostics use transient snapshots, and `didClose` releases retained parsed and semantic state. An initial unbounded workspace-retention design made the full race suite materially slower and memory-heavy; it was rejected before delivery. The retained design restored the focused workspace-limit test to 5.9s and the full `phpstrom` race package to about 37s on the validation host. These are test-run observations, not editor-latency claims.
- Pinned and sibling-development Go tests, vet, and race suites pass. TypeScript lint/compile/package, the VS Code 1.89.1 extension-host suite, all six server target builds, `npm audit --audit-level=low`, and diff checks pass; the audit reports 0 vulnerabilities and packaging retains the known `vscode-languageserver-types` dynamic-require warning.

### 2026-08-30 — Incremental immutable project-index replacement

- Parser commit `f1e06b9` adds `BuildProjectIndexIncremental`: a changed-file copy-on-write path that preserves prior immutable indexes for concurrent readers, refreshes declaration locations, and traverses only changed ASTs. Duplicate/colliding definition cases fall back to the existing sorted full build so winner and diagnostic ordering remain deterministic.
- Extension commit `d6680f2` uses that path for indexed documents, stubs, removals, and unsaved overlays, and pins the exact parser pseudo-version. The semantic-cache revision now advances only when exported classes, functions, methods, properties, or constants change; edits confined to another file's body publish a fresh project view without discarding reusable facts/flow for unchanged documents.
- The checked-in 1,000-file synthetic microbenchmark ran five 500ms samples on the Apple M1 validation host. Fresh-build median was 2.358ms, 3,410,575 B/op, and 37,802 allocs/op; one-file incremental median was 1.115ms, 1,869,680 B/op, and 11,735 allocs/op (52.7% lower time, 45.2% fewer bytes, and 69.0% fewer allocations within the same candidate binary). A separately run pre-change cold-build sample had a 2.405ms median and 3,311,041 B/op; because it was not interleaved, no cold timing improvement is claimed, while the retained source metadata costs about 3.0% in this synthetic allocation measure.
- An earlier contribution-assembly design was rejected before delivery because it raised fresh-build median time to about 4.4ms and allocation to about 7.4 MB/op on the same synthetic shape. Adversarial tests cover full-build equivalence, additions/removals, body/position edits, exported signature changes, duplicate ordering, immutable prior readers, and races.
- Parser tests/vet/race, pinned and sibling extension tests/vet/race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` dynamic-require warning; the audit reports 0 vulnerabilities.

### 2026-08-30 — Dependency-scoped semantic snapshot invalidation

- Parser commit `db625ca` adds deterministic change details to incremental index updates: stable IDs and old/new export records for classes, functions, methods, properties, class constants, and global constants, plus dependency names expanded through previous and current transitive class/interface/trait lineages. Missing source metadata is explicitly incomplete and requires global invalidation.
- Extension commit `70fcf25` pins that engine revision and retains a per-document semantic revision derived from exported-change events. Unrelated exports and body-only edits reuse cached facts/flow; referenced class/member/function/constant changes and transitive base changes rebuild the affected document snapshot while every analysis context still receives the latest immutable resolver.
- Dependency matching is deliberately conservative and source-name-based because generated reference facts are not yet complete. This can invalidate extra documents for common names, but supported resolved calls/types retain a textual owner/member/function/constant reference. Event history is capped at 64 entries and 256 dependency names per event; incomplete, empty, oversized, or compacted histories cause a safe global revision instead of unbounded matching or stale reuse.
- The updated checked-in 1,000-file synthetic benchmark ran five 500ms samples on the Apple M1 validation host. Fresh-build median was 2.483ms; a one-file exported-signature update including dependency reporting was 1.622ms (34.7% lower), and a one-file body-only update was 1.172ms (52.8% lower). The exported update used 20,042 allocations versus 37,802 fresh (47.0% fewer); body-only used 11,727 (69.0% fewer). These are in-process synthetic index measurements, not editor-latency claims.
- Adversarial coverage includes stable change identities, add/remove/rename behavior, deterministic ordering, transitive descendants, missing-metadata fallback, unrelated cache reuse, referenced function/constant/member invalidation, event/name overflow, identifier boundaries, and race-safe immutable readers. Full parser and extension validation passes; packaging retains the known warning and npm audit reports 0 vulnerabilities.

### 2026-08-30 — Trace-based editor-path latency gate

- Parser commit `e97afef` adds an explicit `FullRebuild` result to incremental project-index change reports, distinguishing ordinary changed-file replacement from missing-metadata and definition-collision full fallbacks without inference in the extension.
- Extension commit `d973d68` pins that parser revision and adds bounded 1,024-event handler traces plus atomic index/cache counters. The trace records workspace indexing, scheduled and saved analysis, debounce cancellation, published or stale-dropped diagnostics, full versus incremental builds, body/export changes, global compaction, document revision checks, dependency matches, and parse/semantic cache hits and misses.
- `server/cmd/benchmark-editor` creates a neutral 1,005-file synthetic workspace, runs five workspace starts in fresh Go processes, and drives five fresh incremental scenarios through the real handler/index/provider path. It emits JSON distributions and fails on configurable absolute budgets or missing cache reuse, dependency matching, incremental/body/export updates, cancellation, stale rejection, full fallback, or global compaction. `make test-editor-latency` and the read-only `editor-latency.yml` PR/main gate run it; CI uploads the JSON report even when a budget fails.
- On the Apple M1 validation host, process-cold median was 18.642ms (CV 6.47%). End-to-end on-type medians, including the configured 150ms debounce, were 157.791ms for a body-only edit (CV 1.07%), 159.835ms for a referenced dependency signature edit (CV 0.82%), and 166.799ms for a collision-triggered full fallback (CV 0.14%). Cached save analysis was 0.211ms median; pre-start cancellation was 0.018ms; deterministic stale-result analysis and rejection was 0.197ms. The tiny sub-millisecond samples have higher relative variance and are correctness/absolute-budget evidence, not comparative speed claims.
- The harness begins at fresh Go process startup for cold runs and at handler scheduling for edits. It does not include VS Code/Node extension activation, JSON-RPC transport/serialization, or a representative user workspace, so it is a server editor-path regression gate rather than a complete perceived-editor-latency claim.
- Parser and pinned/sibling extension tests, vet, and race suites pass; all six server targets build; TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the trace gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-30 — Structured UTF-16 analysis ranges

- Extension commit `e4c4c7b` adds a source-position mapper that converts the parser's one-based rune line/column spans into zero-based UTF-16 LSP ranges using the exact analysed source. Valid end positions remain half-open; missing, reversed, or out-of-bounds ends safely collapse to a point at the mapped start.
- Structured analysis findings now consume `AnalysisIssue.EndLine` and `EndColumn` instead of discarding them. Parser-error line/column prefixes also map their start point through the same UTF-16 conversion, while unstructured parser errors retain the `(0,0)` fallback.
- Style diagnostics deliberately remain on their legacy point contract because their coordinate producers are not uniformly parser rune positions. Moving them requires a separate producer-by-producer contract audit rather than applying the analysis conversion blindly.
- Unit tests cover ASCII, BMP characters, astral characters represented by UTF-16 surrogate pairs, multiline spans, CRLF input, missing ends, reversed spans, and invalid/out-of-bounds coordinates. A real level-0 integration case proves that a diagnostic following an emoji on the same line has the expected non-point UTF-16 range.
- Parser and pinned/sibling extension tests, vet, and race suites pass; all six server targets build; TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-30 — Diagnostic-level boundary alignment

- Parser commit `e7ea7cf` moves protected-method visibility on known receivers from level 0 to a dedicated `Level2.MethodVisibility` rule and moves resolved non-`Throwable` objects from level 0 to `Level3.ThrowType`. Calling a static method through valid instance syntax no longer emits the extension's former level-0 false positive.
- Fifteen neutral fixtures expand the level-0 differential pack from 47 to 62 cases. New one-case level-2 and level-3 packs prove the diagnostics are absent below their reference levels and present at the correct boundary. The full 62/1/1 reference run matches pinned PHPStan `2.2.x-dev@e4ab62a` with zero engine or reference mismatches.
- The expanded pack also gates consistent-constructor compatibility, final/abstract modifier parse surfaces, final class constants, interface constant visibility (an explicitly recorded reference divergence), literal increments, missing includes, `printf` placeholders, `$this` in static methods, and additional clean controls. The capability matrix now represents every checked-in level-0 fixture.
- Extension commit `6b4a1a8` pins the exact parser pseudo-version and adds an integration contract proving default editor analysis still surfaces the new level-2 and level-3 codes while instance syntax for static methods remains clean.
- Parser tests, vet, race, and all live differential runs pass. Pinned and sibling extension tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-30 — Typed-receiver unknown method diagnostics

- Parser commit `492e88e` adds `Level2.MethodExistence` for a single resolved class receiver while keeping `$this` ownership in the existing level-0 symbol rule. Typed parameters, direct and assigned `new` expressions, method-return chains, and typed-property chains are covered; known methods, `__call`, mixed receivers, unknown receiver classes, and multi-type receivers remain conservative.
- Eight level-2 fixtures plus one level-0 boundary control expand the executable gates to 63 level-0, 24 level-1, nine level-2, and one level-3 cases. The full 63/9 method reference runs match pinned PHPStan `2.2.x-dev@e4ab62a` with zero engine or reference mismatches.
- Extension commit `b4e2c49` pins pseudo-version `v0.0.0-20260830203247-492e88e949bd`, proves the default editor path surfaces the new diagnostic, and maps the existing undefined-symbol setting to suppress it alongside level-0 symbol diagnostics.
- Parser tests, vet, race, and live level-0/level-2 differential runs pass. Pinned and sibling extension tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-31 — Function, conditional, and multi-class method receivers

- Parser commit `c948b5b` resolves statically named function return types through the project resolver, unions ternary branch inference, and extends `Level2.MethodExistence` to class-only multi-atom receivers when every resolved class lacks the method. The ordinary single-class path remains allocation-light; ambiguous built-in/non-object atoms, unknown classes, and any member providing the method stay conservative.
- Five neutral level-2 fixtures add named function-return, class-only ternary, all-missing union/intersection, and clean intersection-member coverage. The executable gates are now 63 level-0, 24 level-1, fourteen level-2, and one level-3 cases. Complete level-0 and level-2 reference runs match pinned PHPStan `2.2.x-dev@e4ab62a` with zero engine or reference mismatches.
- Extension commit `439bcb0` pins pseudo-version `v0.0.0-20260831064154-c948b5b9b7a6` and proves the default editor path reports exactly the three supported unknown-method forms while union/intersection receivers with an available member remain clean.
- Parser tests, vet, and race pass. Pinned and sibling extension tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-31 — DNF preservation and nullable method receivers

- Parser commit `ad09a14` adds private lossless union-of-intersection metadata while retaining the public flat `Type.String`, `Accepts`, and `SingleClassName` contracts. Balanced outer-parenthesis handling fixes corruption of `(A&B)|(C&D)`; namespace, template-aware, ternary, semantic-fact, cache, and level-0 type-reference paths now preserve or consume DNF members consistently.
- `Level2.MethodExistence` now treats `null` as a removable receiver alternative, reporting the class method absence for nullable parameters and class-or-null ternaries while known nullable methods remain clean. Pinned PHPStan level 2 stays clean when any DNF member provides the method; higher-level per-alternative enforcement is deliberately not pulled down into this rule.
- Five neutral fixtures expand level 2 from fourteen to nineteen cases: all-missing DNF, clean available-member DNF, nullable missing/known, and nullable-ternary missing. The full 63-case level-0 and 19-case level-2 reference runs match pinned PHPStan `2.2.x-dev@e4ab62a` with zero engine or reference mismatches.
- Extension commit `5ea6c4a` pins pseudo-version `v0.0.0-20260831071512-ad09a14bf44f` and proves the default editor path has no false level-0 DNF symbol diagnostic while surfacing the three supported DNF/nullable level-2 findings.
- Parser tests, vet, and race pass. Pinned and sibling extension tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-31 — Callable returns and dynamic class-string construction

- Parser commit `b2e5b7a` retains concrete return targets for PHPDoc callable parameters and assigned closures/arrow functions inside immutable function scopes. Invoking those variables now produces self-contained inferred receiver facts without changing the public flat `Type` contract; reassignment clears stale callable metadata and clone writes remain sibling-safe.
- Concrete PHPDoc `class-string<T>` parameters now retain their normalized target for `new $class()` inference. Dynamic construction no longer creates a false level-0 missing-class diagnostic, while unresolved, reassigned, malformed, multi-argument, union, and template-valued class strings remain conservative.
- PHPDoc parameter splitting now preserves standard spaced callable signatures such as `callable(): Service`. Six neutral missing/known fixtures expand level 2 from nineteen to twenty-five cases; the complete 63-case level-0 and 25-case level-2 runs match pinned PHPStan `2.2.x-dev@e4ab62a` with zero engine or reference mismatches.
- Extension commit `b8bbc27` pins pseudo-version `v0.0.0-20260831123607-b2e5b7abcd9e` and proves the default editor path reports exactly the three callable/closure/class-string method findings while known controls and level-0 symbols remain clean.
- A scheduled full-corpus run exposed a scope-less property-fetch panic in the level-2 fallback walker. Parser commit `97c5e60` makes that path conservative when no function scope exists; a local full PSL analysis completed 7,259/7,259 files with zero parse failures, and extension commit `cef31c7` advances the production pin to the hardened revision.
- Parser tests, vet, and race pass. Pinned and sibling extension tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-31 — Declared callable, array-shape, and template class-string receivers

- Parser commit `fe4dd3d` retains callable-return metadata on indexed properties, methods, interface members, and global functions, and resolves `@template T of Class` bounds for `class-string<T>` construction.
- This follow-up extracts literal PHPDoc `array{key: callable(): T}` fields into copy-on-write function scopes so assigned, copied, and directly invoked shape elements produce the same receiver facts. Unknown keys, non-callable fields, and dynamic indexes remain conservative.
- Eight missing/known fixtures expand level 2 from twenty-five to thirty-three cases; the complete 63-case level-0 and 33-case level-2 runs match pinned PHPStan `2.2.x-dev@e4ab62a` with zero engine or reference mismatches.
- Native property type hints keep precedence over non-callable `@var` generics, and imported generic PHPDoc names such as `Collection<string, Policy>` keep their use aliases.
- Parser tests, vet, and the analyser race suite pass. Extension commit `b40b003` pins pseudo-version `v0.0.0-20260831132841-badc81c2a8e8` and proves the default editor path reports exactly the four declared-callable, array-shape, and template class-string findings while known controls and level-0 symbols remain clean. Pinned and sibling extension tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-31 — Nested shapes and remaining object-expression receivers

- Nested PHPDoc `array{inner: array{key: callable(): T}}` fields and `list{callable(): T}` elements retain copy-on-write shape metadata through assignment and chained access.
- `clone`, `??`, `match`, and nullsafe method calls infer object receivers at the pinned level-2 boundary.
- Parser commit `f3095cd` lands the nested/list shape and object-expression inference. Eight fixtures expand level 2 from thirty-three to forty-one cases; the complete 63-case level-0 and 41-case level-2 runs match pinned PHPStan `2.2.x-dev@e4ab62a` with zero engine or reference mismatches. Scalar/array/`object` receivers remain a separate `method.nonObject` identifier.
- Parser tests, vet, and the analyser race suite pass. Extension commit `82f2a08` pins pseudo-version `v0.0.0-20260831134345-f3095cd1086e` and proves the default editor path reports exactly the six nested-shape, list, clone, coalesce, match, and nullsafe findings while known nested-shape methods and level-0 symbols remain clean. Pinned and sibling extension tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 extension-host tests, the editor-latency gate, and `npm audit --audit-level=low` pass. Packaging retains the known `vscode-languageserver-types` warning and the audit reports 0 vulnerabilities.

### 2026-08-31 — Process-cold benchmark stability protocol

- `cmd/benchmark` pins worker `GOMAXPROCS` to `--workers`, discards unmeasured process-cold warmups, settles between subprocesses, and can append extra measured runs when CV exceeds 5% without dropping outliers from the gate. Drop-max CV is recorded only as a diagnostic. Linux reports also include `/proc/loadavg`.
- Weekly CI now uses Mago-aligned path sets for the three required workloads and `--max-cv 0` so noisy hosted runners still upload accounting and RSS artifacts. Local interleaved comparisons keep the 5% gate.
- A WordPress indicator on this host accounted for 5,357/5,357 files, 1,451,208 LOC, 47,344,277 bytes, zero parse failures, and 24,770 diagnostics on every run, then remained rejected at 8.22% CV after twenty measured cold runs. See `docs/benchmarks/2026-08-31-wordpress-stability-protocol.md`.
- PHP Strom's synthetic editor harness discards one process-cold warmup and pins worker `GOMAXPROCS`.

### 2026-08-31 — Level-2 dynamic array-shape and list indexes

- Eight fixtures expand level 2 from forty-nine to fifty-seven cases for assigned constant keys, concatenated literals, class constants, unknown string indexes (union of callable fields), unknown int list indexes, and matching known-method controls. Complete 63-case level-0 and 57-case level-2 runs match pinned PHPStan `2.2.x-dev@e4ab62a`. Int indexes into purely named shapes and unknown literal keys stay conservative. Per-alternative DNF remains a separate higher reference level.
- Parser commit `6372f1d` lands the inference. Extension commit `5462c74` pins pseudo-version `v0.0.0-20260831144126-6372f1de78af` and proves the default editor path reports assigned, concatenated, class-constant, and unknown int-list missing methods while known assigned and unknown-string methods stay clean. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and `npm audit --audit-level=low` pass.

### 2026-08-31 — Level-2 method calls on non-object receivers

- Eight fixtures expand level 2 from forty-one to forty-nine cases for `int`/`array`/`callable`/`iterable`, clean `object`, and class-or-scalar unions. Complete 63-case level-0 and 49-case level-2 runs match pinned PHPStan `2.2.x-dev@e4ab62a`.
- Parser commit `6df9189` lands the rule. Extension commit `8dacb2b` pins pseudo-version `v0.0.0-20260831141306-6df9189dbf3d`, maps the diagnostic to type-error toggles, and proves the default editor path reports the six non-object findings while `object` and known class-or-string methods stay clean. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and `npm audit --audit-level=low` pass.

### 2026-08-31 — Roadmap tidy

- Living docs now treat M0 as the working baseline and M1 as in-progress. Ranked next work is leftover level-2 expression receivers, ungated level-0 unit-tested surfaces, then higher-level DNF. Performance stays measurement-only until an isolated-host interleaved run passes the 5% CV contract. `FEATURES.md` architecture now names the Go server; leftover TypeScript/tree-sitter prose is marked obsolete.

### 2026-08-31 — Remaining level-2 expression-form receivers

- Eight fixtures expand level 2 from fifty-seven to sixty-five cases for file-level constants, other same-file class constants, `match` indexes, property `@var` shapes, method `@return` shapes, and `list{Class}` object indexes, plus known-method `match` and property controls. Complete 63-case level-0 and 65-case level-2 runs match pinned PHPStan `2.2.x-dev@e4ab62a`. Static properties are unit-tested. Per-alternative DNF remains a separate higher reference level.
- Parser commit `ab99bf5` lands the inference. Extension commit `296d585` pins pseudo-version `v0.0.0-20260831151502-ab99bf53c3f0` and proves the default editor path reports global-const, foreign-class-const, match, property, method-return, and list-object missing methods while known match and property methods stay clean. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and `npm audit --audit-level=low` pass.

### 2026-08-31 — Expand the level-0 pack to eighty cases

- Seventeen reviewed fixtures expand level 0 from sixty-three to eighty cases for unknown parents, invalid implements/extends/trait-use combinations, non-public interface implementations, inherited parameter/return compatibility, enum case legality, static calls to instance methods, duplicate named arguments, unknown function imports, and unknown static methods, plus a covariant-return clean control. Complete 80-case level-0 and 65-case level-2 runs match pinned PHPStan `2.2.x-dev@e4ab62a`. Left out: inherited parameter-name changes (PHPStan silent), extra-required-parameter overrides, remaining enum constructor/backing/`Serializable` cases, and `(void)`/`(unset)` casts.
- Parser commit `a148d43` lands the pack. Extension commit `c19fd50` pins pseudo-version `v0.0.0-20260831152801-a148d43be45e` and proves the default editor path reports implementing a class, extending an interface, unit-enum values, static calls to instance methods, and unknown static methods. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and `npm audit --audit-level=low` pass.

### 2026-08-31 — Gate leftover level-0 enum, signature, and cast surfaces

- The parser now accepts `(void)` and `(unset)` casts so the existing language rule can report them. Eight fixtures expand level 0 from eighty to eighty-eight cases for extra required parameters, enum constructor/destructor/magic/`Serializable`/float backing, and void/unset casts. Complete 88-case level-0 runs match pinned PHPStan `2.2.x-dev@e4ab62a`. Left out: inherited parameter-name changes (PHPStan silent) and native enum method redeclaration (mixed extra identifiers).
- Parser commit `61f2487` lands the pack. Extension commit `a782887` pins pseudo-version `v0.0.0-20260831153651-61f24873c7dd` and proves the default editor path reports void/unset casts and enum constructors. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and `npm audit --audit-level=low` pass.

### 2026-08-31 — Detect unknown methods on partial unions at level 7

- `Level7.MethodUnion` reports when some but not all DNF alternatives provide the method, matching PHPStan `method.notFound`. All-missing unions stay on the level-2 rule. Five fixtures bring the executable gates to 88/24/65/1/5 and match pinned PHPStan `2.2.x-dev@e4ab62a`. Nullable `method.nonObject` stays at level 8.
- Parser commit `a92ff39` lands the pack. Extension commit `960f927` pins pseudo-version `v0.0.0-20260831160450-a92ff395870a` and proves the default editor path reports partial unions through the undefined-symbol setting. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and `npm audit --audit-level=low` pass.

### 2026-08-31 — Detect known methods on nullable object receivers at level 8

- `Level8.MethodNonObject` reports when every remaining object-like alternative provides the method and `null` is also possible, matching PHPStan `method.nonObject`. Scalar `int|null` and unknown nullable methods stay on level 2. Nullsafe `?->` stays clean. Five fixtures bring the executable gates to 88/24/65/1/5/5 and match pinned PHPStan `2.2.x-dev@e4ab62a`.
- Parser commit `e629d42` lands the pack and records nullsafe operators on `MethodCallNode`. Extension commit `8348748` pins pseudo-version `v0.0.0-20260831163118-e629d4267f4c` and proves the default editor path reports known nullable methods through the type-error setting. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and `npm audit --audit-level=low` pass.

### 2026-08-31 — Snapshot-backed full analysis and profile-driven speed work

- `cmd/benchmark` and `analyze` now share one `SemanticSnapshot` per full run. Parser commit `3155c07` also combines the level-2/7/8 method-receiver walks, reuses file/namespace type context, folds ASCII identifier keys, and scans empty-statement source without splitting the file into cached lines. Follow-up `abb6633` visits level-0 type, symbol, and language checks in one walk. Method-visibility, throw-type, and return-type checks now share one AST walk when the analysis level includes them. Deprecated-call issues are collected during the shared argument walk. WordPress still accounts for 5,357/5,357 files and 26,321 diagnostics on the profile path. No Mago comparison is claimed.

### 2026-08-31 — Compact semantic facts and reuse the argument walk

- Semantic facts are partitioned by filename internally, so each stored span no longer repeats the filename in both its map key and value. `asciiLowerIdent` now returns its owned lowercase byte buffer without a second string copy, with allocation regression coverage. Method-receiver diagnostics are collected during the existing flow-sensitive argument walk while preserving file-scope and unsupported-shape fallbacks.
- A three-iteration WordPress profile retained 5,357/5,357 files and 26,321 diagnostics while allocated space fell from 8.43 GB at exact baseline `de5598e` to 6.46 GB, a 23.3% reduction. Generated-fact insertion fell from approximately 1.54 GB to 0.75 GB in the sampled profile.
- A ten-round exact-baseline cold comparison retained identical accounting and produced candidate/baseline means of 2.964s/3.435s and maximum RSS of 1.357GB/1.665GB. It is rejected as a performance claim because candidate CV was 5.15% versus the 5% contract.
- A separate ten-round Mago 1.47.4 indicator produced raw engine/Mago mean ratios of 0.848x and maximum-RSS ratios of 1.152x, but both CVs failed and semantic/file accounting remained non-comparable. See `docs/benchmarks/2026-08-31-wordpress-snapshot-allocation-batch.md`; no Mago-parity claim is made.

### 2026-08-31 — Persistent function-scope type layers

- Variable and property flow types now use bounded persistent layers: clones share immutable parents, first writes use inline one-entry deltas, and longer chains compact at a fixed depth. This replaces eager whole-map detachment while preserving original, child, sibling, cached-class-property, and chained-scope isolation. Missing callable/array-shape deletion also avoids needless copy-on-write detachment.
- A three-iteration WordPress profile retained 5,357/5,357 files and 26,321 diagnostics while allocated space fell from 6.61 GB at exact baseline `6fdfba0` to 5.54 GB, a 16.2% reduction. The prior approximately 1.04 GB `copyTypeMap` hot site disappeared.
- The ten-round interleaved process-cold comparison passed: candidate/baseline means were 2.432s/2.511s with CVs 4.26%/1.87%, medians 2.392s/2.493s, and maximum RSS 1.408GB/1.489GB. Accounting was identical in every run. This supports a 3.1% exact-baseline mean improvement and 5.4% maximum-RSS reduction, not a Mago-parity claim.
- An assignment-observer shortcut was rejected before delivery because it changed WordPress diagnostics by 42. See `docs/benchmarks/2026-08-31-wordpress-persistent-scope-layers.md`.

### 2026-08-31 — Allocation-light function resolver views

- Private analyser function lookups now borrow immutable index-owned parameter metadata, while the public `SemanticSnapshot.ResolveFunction` API retains defensive parameter copies. Focused tests cover public mutation isolation, internal backing-storage reuse, and lower per-call allocation.
- A three-iteration WordPress profile retained 5,357/5,357 files and 26,321 diagnostics while allocated space fell from 5.54 GB at exact baseline `0f8eee4` to 5.11 GB, a 7.8% reduction. The prior approximately 0.44 GB `SemanticSnapshot.ResolveFunction` hot site disappeared.
- The ten-round interleaved comparison was stable but runtime-neutral: candidate/baseline means were 2.434s/2.431s with CVs 2.00%/3.54%; maximum RSS was 1.347GB/1.366GB. See `docs/benchmarks/2026-08-31-wordpress-function-resolver-view.md`; no speed or Mago-parity claim is made.

### 2026-09-01 — Remove duplicate method-parameter copies

- Snapshot method and own-method resolution now relies on the already-defensive `ProjectIndex` result instead of cloning parameter slices a second time. Focused mutation and allocation tests prove the public immutable boundary remains intact and the snapshot facade adds no copy.
- A three-iteration WordPress profile retained 5,357/5,357 files and 26,321 diagnostics while allocated space fell from 5.11 GB at exact baseline `cbb469b` to 4.93 GB, a 3.5% reduction. The prior approximately 0.21 GB `SemanticSnapshot.ResolveMethod` hot site disappeared.
- The ten-round cold comparison is rejected because baseline CV was 5.33%. The raw candidate/baseline means of 2.703s/2.797s and RSS samples are not accepted performance evidence. See `docs/benchmarks/2026-09-01-wordpress-method-param-copy.md`.

### 2026-09-01 — Function-scope metadata copy-on-write

- Branch clones now share array-index and generic-instance maps until the first real write. Ingress and lookup boundaries retain slice isolation, missing-key clears preserve sharing, and generic maps are allocated lazily.
- A three-iteration WordPress profile retained 5,357/5,357 files and 26,321 diagnostics while allocated space fell from 4.93 GB at exact baseline `1696897` to 4.75 GB, a 3.7% reduction. Eager generic-context copying disappeared; array-index copying now occurs only on actual first writes.
- The ten-round exact-baseline gate passed: candidate/baseline means were 2.667s/2.773s with CVs 3.17%/4.08%, medians 2.628s/2.738s, and maximum RSS 1.291GB/1.295GB. This supports a 3.8% mean, 4.0% median, and 0.3% maximum-RSS improvement, not a Mago-parity claim. See `docs/benchmarks/2026-09-01-wordpress-scope-metadata-cow.md`.

### 2026-09-01 — Allocation-light method resolver views

- Internal existence, visibility, argument-count, and snapshot symbol-ID lookups now borrow immutable index-owned method metadata. Public `ResolveMethod` still clones parameters and rewrites inherited generic bindings. Type inference is unchanged.
- A three-iteration WordPress profile retained 5,357/5,357 files and 22,387 diagnostics while allocated space fell from 4.77 GB at exact baseline `9a4f4a4` to 4.50 GB, a 5.7% reduction. The previous approximately 0.79 GB public `ResolveMethod` path shrank to 0.23 GB.
- The twenty-round interleaved comparison is rejected: candidate/baseline CVs were 26.9%/9.0%. Accounting was identical. See `docs/benchmarks/2026-09-01-wordpress-method-resolver-view.md`.

### 2026-09-01 — Snapshot insertion, compact CFG, identifier intern

- Narrowing facts write straight into the per-file store. Linear and loop graphs keep one- and two-successor edges inline, store reachability once, and share parsed statement slices with variable-flow construction. Remaining identifier `ToLower` paths use `asciiLowerIdent`, and mixed-case ASCII results are interned.
- A three-iteration WordPress profile retained 5,357/5,357 files and 22,387 diagnostics while allocated space fell from 4.49 GB on the method-view working tree to 3.67 GB, an 18.3% reduction. The previous 0.87 GB identifier-fold site left the hot list.
- The fourteen-sample interleaved comparison against `9a4f4a4` passed: candidate/baseline means were 2.392s/2.601s with CVs 4.97%/3.04%, medians 2.354s/2.568s, and maximum RSS 1.206GB/1.321GB. That 8.0% mean improvement includes the uncommitted method-resolver views. See `docs/benchmarks/2026-09-01-wordpress-snapshot-intern.md`.

### 2026-09-01 — Compact semantic-fact keys

- Built-in fact kinds use separate per-file maps with packed ordinary source spans, while generated inferred facts omit the external-value field. Custom kinds, caller-supplied values, large offsets, exact lookup, deterministic enumeration, and duplicate handling keep their existing contracts.
- A three-iteration WordPress profile retained 5,357/5,357 files and 22,387 diagnostics while allocated space fell from 3,873,268,076 bytes at exact baseline `cfc1c50` to 3,516,256,187 bytes, a 9.2% reduction. Semantic-fact insertion fell 44.3%, from 783,376,032 to 435,973,845 bytes.
- The twenty-round interleaved comparison is rejected: candidate/baseline CVs were 9.00%/23.44%. Accounting was identical. See `docs/benchmarks/2026-09-01-wordpress-semantic-fact-keys.md`; no cold-speed or RSS claim is made.
- Parser commit `8f400be` lands the storage change. Extension commit `59eb0b0` pins pseudo-version `v0.0.0-20260901182711-8f400be53b88`. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and a zero-vulnerability npm audit pass.

### 2026-09-01 — Shared immutable function-scope context

- Function-scope clones now share immutable class and file-type context through one pointer while retaining copy-on-write isolation for every mutable state family. The shallow scope value fell from 208 to 96 bytes; read-only clone time across five focused samples fell from 43.74–44.52ns to 26.68–26.97ns.
- A three-iteration WordPress profile retained 5,357/5,357 files and 22,387 diagnostics while allocated space fell from 3,516,256,187 bytes at exact baseline `8f400be` to 3,273,965,861 bytes, a 6.9% reduction. `functionScope.clone` fell 56.7%, from 401,159,880 to 173,555,216 bytes.
- The twenty-round interleaved comparison is rejected: candidate/baseline CVs were 23.39%/18.04%. Accounting was identical. See `docs/benchmarks/2026-09-01-wordpress-function-scope-context.md`; no cold-speed or RSS claim is made.
- Parser commit `f98411e` lands the shared context. Extension commit `ad89ed1` pins pseudo-version `v0.0.0-20260901185820-f98411ef2d87`. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and a zero-vulnerability npm audit pass.

### 2026-09-01 — Compact control-flow storage

- Flow graphs and reachability are partitioned by file with packed ordinary spans, compact built-in kinds, one reachability-state map, local block offsets, an inline first graph, and dense storage for additional graphs. Custom kinds, large offsets, ambiguity handling, parent resolution, and defensive public graph copies retain their contracts.
- A three-iteration WordPress profile retained 5,357/5,357 files and 22,387 diagnostics while allocated space fell from 3,273,965,861 bytes at exact baseline `f98411e` to 3,046,140,014 bytes, a 7.0% reduction. The combined graph/reachability sites fell 50.2%, from 609,688,347 to 303,490,806 bytes.
- The ten-round exact-baseline gate passed: candidate/baseline means were 2.017s/2.141s with CVs 3.11%/1.97%, medians 2.024s/2.147s, and maximum RSS 1.023GB/1.076GB. This supports a 5.8% mean, 5.7% median, and 4.9% maximum-RSS improvement. See `docs/benchmarks/2026-09-01-wordpress-compact-flow-storage.md`.
- Parser commit `41dfb55` lands the compact store. Extension commit `fa2ce9a` pins pseudo-version `v0.0.0-20260901192221-41dfb5580c97`. Pinned and sibling tests, vet, race, all six server builds, TypeScript lint/compile/package, VS Code 1.89.1 host tests, the editor-latency gate, and a zero-vulnerability npm audit pass.

### 2026-09-01 — Stable same-machine Mago resource comparison

- Ten process-cold rounds alternated the production Go pipeline with verified-current Mago 1.47.4 on the same Apple M1 host. Both full-sample CVs passed: Go 2.61%, Mago 3.64%.
- Go mean time was 2.059s versus Mago's 3.620s (0.569x); maximum peak RSS was 1.061GB versus 1.081GB (0.982x). This passes the 1.5x mean / 1.25x RSS envelope and the equal-or-faster time stretch target.
- Go retained exact 5,357/5,357-file and 22,387-diagnostic accounting. Mago listed 3,183 primary files, used 2,174 vendor PHP files as configured dependency includes, and emitted a stable 218,741 findings. The resource comparison is accepted for these explicitly different current rule sets; semantic parity is not claimed. See `docs/benchmarks/2026-09-01-wordpress-mago-1.47.4-comparison.md`.

## Next ranked candidates

The analyser target in `docs/full-static-analyser-target.md` is the source of truth for ordering.

1. **Performance:** capture a new production profile before any further optimization and only tune a new measured leader; the same-machine Mago resource envelope now passes.
2. **Maintenance:** rewrite `FEATURES.md` for the Go language server and decide the fate of `src/server`.
3. **Source mapping:** structured parser errors, then style-rule coordinate producers currently stuck at points.
4. **Dependency matching:** replace conservative lexical invalidation only after generated reference facts cover supported resolver paths.
