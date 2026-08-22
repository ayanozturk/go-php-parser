# Default VS Code PHP Platform Progress

Last updated: 2026-08-21 (Europe/London)

This file records reproducible evidence for the cooperating `go-php-parser` engine and `vscode-php-strom` extension. PHPStan, PHPCS, and Mago are benchmark references only; no parity claim is made.

## Current baseline

- Engine checkout: `/Users/ayan/Projects/go-php-parser`, `main` at pushed engine commit `8799ed1` before this progress update.
- Extension checkout: `/Users/ayan/Projects/vscode-php-strom`, `main` at pushed integration commit `23dc399` before this security update; the validated release target is `0.1.23`.
- Production extension builds pin `github.com/ayanozturk/go-php-parser` at pseudo-version `v0.0.0-20260820125828-8799ed160392`; `make test-server-dev` validates the sibling engine checkout through a generated, ignored Go workspace.
- Go toolchain observed: Go 1.26.2. Node toolchain observed: Node 22.20.0 and npm 11.7.0.
- Checked-in representative corpus: 32,990 PHP files across Composer, Drupal, Laravel, PHPUnit, and Symfony under `test_projects` (428 MB on disk, including installed dependencies).
- `go run ./cmd/compat-metrics -root test_projects -workers 4 -top 2`: 94.28% file compatibility (31,102 passing, 1,888 failing), 156,002 parse errors, 2.779s. Per project: Composer 98.15%, Drupal 96.37%, Laravel 92.00%, PHPUnit 90.60%, Symfony 93.47%.

## Security and fuzzing

- Clean `npm ci` plus `npm audit --audit-level=low`: 0 known vulnerabilities across 531 installed dependencies on 2026-08-20.
- GitHub remote verification after push: 0 open Dependabot alerts, 33 fixed Dependabot alerts, and 0 open secret-scanning alerts. The push-time banner briefly reported 26 alerts while GitHub recalculated the default branch; the alerts API is the recorded final state.
- `govulncheck`, `gosec`, `staticcheck`, and `golangci-lint` were not installed; no result is claimed for them.
- Neither Go repository currently has a `Fuzz...` target. Malformed/untrusted PHP fuzzing is therefore not continuously exercised.
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

## Maintainability and architecture risks

- `FEATURES.md` still describes a TypeScript/tree-sitter server architecture, while production editor features now use the Go language server and `go-php-parser`. This makes feature/architecture claims hard to audit.
- The obsolete `src/server` TypeScript implementation is excluded from both the production TypeScript build and ESLint; retaining dead server code remains an architecture/deletion decision requiring separate evidence.
- The parser API still exposes start positions rather than complete byte spans, limiting reliable Unicode-aware diagnostics, rename, references, and selection ranges.

## Coverage gaps

- PHPStan benchmark: level 0 remains partial; levels 1 and 2 are largely uncovered; level 3 return/property checks are partial. Missing areas include control-flow precision, arbitrary-expression method checks, PHPDoc validation, broader built-in signatures, and complete modern-syntax coverage.
- PHPCS benchmark: the repository comparison records 16 style rules, far below the breadth of PHPCS standards. Security-oriented source rules such as eval/backtick/forbidden-function checks are not implemented.
- PHP version/framework coverage is represented by the five-project corpus, but the 1,888 failing files demonstrate substantial parser-recovery gaps. Blade/mixed-template files are included in the failures and should be classified separately from pure-PHP parser failures.
- Extension integration gaps include stale architecture documentation and no reproducible cold-start/incremental-edit/cancellation benchmark suite. Workspace-discovery symlink/special-file coverage is now present for the supported macOS/Linux security boundary.

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

## Next ranked candidates

1. **Security:** add deterministic parser/indexer fuzz targets with malformed PHP seeds, panic-error assertions, cancellation checks, and a documented short CI fuzz budget.
2. **Performance:** create repeatable cold/warm index and incremental-edit benchmarks with stable corpus/configuration metadata and peak-RSS/allocation capture.
3. **Maintenance:** correct `FEATURES.md` to the production architecture and decide whether the excluded legacy TypeScript server should be removed.
4. **Features:** after the higher-priority gates, classify and minimize the largest pure-PHP corpus failures before expanding PHPStan/PHPCS diagnostic breadth.
