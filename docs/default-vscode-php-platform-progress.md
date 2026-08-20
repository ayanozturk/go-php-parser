# Default VS Code PHP Platform Progress

Last updated: 2026-08-20 (Europe/London)

This file records reproducible evidence for the cooperating `go-php-parser` engine and `vscode-php-strom` extension. PHPStan, PHPCS, and Mago are benchmark references only; no parity claim is made.

## Current baseline

- Engine checkout: `/Users/ayan/Projects/go-php-parser`, `main` at pushed feature commit `1bdc065` before this evidence-only progress update.
- Extension checkout: `/Users/ayan/Projects/vscode-php-strom`, `main` at pushed commit `f3d205b`.
- Production extension builds pin `github.com/ayanozturk/go-php-parser` at pseudo-version `v0.0.0-20260820080800-1bdc0650cf06`; `make test-server-dev` validates the sibling engine checkout through a generated, ignored Go workspace.
- Go toolchain observed: Go 1.26.2. Node toolchain observed: Node 22.20.0.
- Checked-in representative corpus: 32,990 PHP files across Composer, Drupal, Laravel, PHPUnit, and Symfony under `test_projects` (428 MB on disk, including installed dependencies).
- `go run ./cmd/compat-metrics -root test_projects -workers 4 -top 2`: 94.28% file compatibility (31,102 passing, 1,888 failing), 156,002 parse errors, 2.779s. Per project: Composer 98.15%, Drupal 96.37%, Laravel 92.00%, PHPUnit 90.60%, Symfony 93.47%.

## Security and fuzzing

- Clean `npm ci` plus `npm audit --audit-level=low`: 0 known vulnerabilities across 531 installed dependencies on 2026-08-20.
- GitHub remote verification after push: 0 open Dependabot alerts, 33 fixed Dependabot alerts, and 0 open secret-scanning alerts. The push-time banner briefly reported 26 alerts while GitHub recalculated the default branch; the alerts API is the recorded final state.
- `govulncheck`, `gosec`, `staticcheck`, and `golangci-lint` were not installed; no result is claimed for them.
- Neither Go repository currently has a `Fuzz...` target. Malformed/untrusted PHP fuzzing is therefore not continuously exercised.
- The parser recovers panics into `Parser panic:` errors and supports cooperative context cancellation. The language-server indexer applies a 20-second per-file parse timeout and a four-worker cap.
- Before the 2026-08-19 change, the indexer used `os.ReadFile` before enforcing `files.maxSize`, so the configured gate did not bound allocation for oversized or growing files.
- Remaining concrete file-boundary risk: `phpstrom.readDocumentTextFromDisk` still reads an LSP-provided file URI with unbounded `os.ReadFile`; workspace/symlink boundary semantics are not covered by adversarial tests.

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

## Maintainability and architecture risks

- `FEATURES.md` still describes a TypeScript/tree-sitter server architecture, while production editor features now use the Go language server and `go-php-parser`. This makes feature/architecture claims hard to audit.
- The obsolete `src/server` TypeScript implementation is excluded from both the production TypeScript build and ESLint; retaining dead server code remains an architecture/deletion decision requiring separate evidence.
- The parser API still exposes start positions rather than complete byte spans, limiting reliable Unicode-aware diagnostics, rename, references, and selection ranges.

## Coverage gaps

- PHPStan benchmark: level 0 remains partial; levels 1 and 2 are largely uncovered; level 3 return/property checks are partial. Missing areas include control-flow precision, arbitrary-expression method checks, PHPDoc validation, broader built-in signatures, and complete modern-syntax coverage.
- PHPCS benchmark: the repository comparison records 16 style rules, far below the breadth of PHPCS standards. Security-oriented source rules such as eval/backtick/forbidden-function checks are not implemented.
- PHP version/framework coverage is represented by the five-project corpus, but the 1,888 failing files demonstrate substantial parser-recovery gaps. Blade/mixed-template files are included in the failures and should be classified separately from pure-PHP parser failures.
- Extension integration gaps include missing adversarial file-boundary tests, stale architecture documentation, and no reproducible cold-start/incremental-edit/cancellation benchmark suite.

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

## Next ranked candidates

1. **Security:** bound and workspace-validate `readDocumentTextFromDisk` for save/closed-file diagnostics, including oversized files, encoded file URIs, symlink targets, and non-file schemes.
2. **Security:** add deterministic parser/indexer fuzz targets with malformed PHP seeds, panic-error assertions, cancellation checks, and a documented short CI fuzz budget.
3. **Performance:** create repeatable cold/warm index and incremental-edit benchmarks with stable corpus/configuration metadata and peak-RSS/allocation capture.
4. **Maintenance:** correct `FEATURES.md` to the production architecture and decide whether the excluded legacy TypeScript server should be removed.
5. **Features:** after the higher-priority gates, classify and minimize the largest pure-PHP corpus failures before expanding PHPStan/PHPCS diagnostic breadth.
