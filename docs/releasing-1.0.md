# Preparing a 1.0 release

This document tracks stabilization for the parser, analyzer, CLI, and PHP Strom integration as one supported system. A successful Linux build or passing unit suite alone is not a 1.0 release signal. Do not publish a tag or binaries until the required checks below have evidence attached to the release candidate.

## Release gates

| Gate | Required evidence | Current state |
| --- | --- | --- |
| Build and CLI contract | `make build` creates `tusk` (`tusk.exe` on Windows); documented commands, current-directory config discovery, output formats, and exit codes match command tests | Passed locally: full tests and vet; native and Windows executable builds; smoke runs for init, style, analyze, AST, tokens, config, file listing, style-rule listing, source invocation, and Make invocation |
| Go API and module contract | Supported Go toolchain range, public packages/symbols, compatibility policy, module path, and dependency licenses reviewed | Contract documented in [compatibility-1.0.md](compatibility-1.0.md); dependency license review remains open |
| Platform support | CI builds every declared OS/architecture and runs tests on declared native validation hosts; release archives are reproducible and contain the expected executable and license | CI passed at `82b69e45`: all six OS/architecture builds plus Linux/amd64 tests and vet ([Test run](https://github.com/ayanozturk/go-php-parser/actions/runs/37450678558)). Native execution on every target and release archive reproducibility remain open |
| Parser correctness | Lossless identity, gold trees, Zend token parity, corpus identity/accounting, and malformed-input fuzz gates pass | Passed locally on Go 1.23.12/Linux amd64: full tests, race suite, lexer token parity, all four 2-minute fuzz smoke runs, and identity/accounting at 100% for Symfony (10,028/10,028 files; 80,347,084 bytes) and WordPress (3,188/3,188 files; 36,103,588 bytes). CI parser/PHPDoc fuzz jobs also passed at `82b69e45` ([Security fuzz run](https://github.com/ayanozturk/go-php-parser/actions/runs/37450678621)) |
| Analyzer correctness | Full tests, race suite, exact differential fixtures, reviewed-corpus quality thresholds, and complete corpus accounting pass | Strict pinned fixtures pass: 507 supported cases matched exactly; one unsupported level 0 case retains its reference diagnostic. The 1,775-file PSL source corpus has complete file accounting but fails diagnostic quality review, with thousands of unresolved differences. See [reviewed corpus quality](reviewed-corpus-quality.md), [analyser capability matrix](analyser-capability-matrix.md), and roadmap release gate |
| CLI and PHP Strom integration | Installation/config migration guide, exit-code contract, supported integration version, and pinned-module plus sibling-development tests pass | Open; see [PHP Strom integration review](phpstrom-integration-review.md) |
| Performance and resources | Every change preserves or improves our best accepted process-cold CLI and engine results: at least 10 repetitions on identical pinned workloads, CV ≤5%, complete file/diagnostic parity, and no time or peak-memory regression beyond measurement noise | Open; establish fresh best-result baselines for the release workload set, then maintain the rolling record |
| Release package | Changelog, known limitations, migration guidance, dependency/license notices, checksums, and rollback instructions reviewed | Open |

## Candidate validation commands

Run from a clean checkout of the candidate commit. Save complete command output and tool versions with the release evidence.

```sh
go version
go test ./...
go vet ./...
go test -race ./...
go test ./syntax -run '^$' -fuzz '^FuzzSyntaxParserMalformedPHP$' -fuzztime 2m
go test ./syntax -run '^$' -fuzz '^FuzzSyntaxIndexParserMalformedPHP$' -fuzztime 2m
go test ./ast -run '^$' -fuzz '^FuzzParsePHPDoc$' -fuzztime 2m
go test ./analyse -run '^$' -fuzz '^FuzzParseType$' -fuzztime 2m
make build
./tusk --help
```

### Latest correctness-gate evidence

On 2026-10-06, Go 1.23.12 on Linux/amd64 passed `go test ./...`, `go vet ./...`,
and `go test -race ./...`. The uncached
`go test -count=1 ./cmd/diagnostic-diff ./lexer` passed. Four two-minute fuzz
runs passed: the full syntax parser, declaration-tier syntax parser, PHPDoc
parser, and type parser. The Symfony and
WordPress lossless identity and accounting reports both had zero failing files
and 100% source identity. Local `go build ./...` cross-builds passed for Linux,
macOS, and Windows on both amd64 and arm64. The Test and Security fuzz workflows
passed for commit `82b69e45` (linked in the release-gate table above).

The CLI's built-in `--help` output lists Go flags; command names and descriptions are verified by `go test ./command`. Smoke-test config initialization in a temporary working directory (`tusk init` must create `tusk.yaml`) and run representative style, AST, token, config, file-list, and analysis commands against an explicit `tusk.yaml` with `-config` before the command.

The fuzz commands are scheduled smoke checks, not exhaustive proof. Also run the corpus identity, differential, PHPStan comparison, PHP Strom integration, cross-platform, and performance workloads required by the rows above. Record exact corpus commits, configuration, Go version, run counts, file/byte totals, parse failures, diagnostics, peak RSS, coefficient of variation, and artifact hashes. Do not turn a missing or unsupported workload into a passing result.

## Version and publication sequence

1. [x] Freeze the release scope and supported compatibility contract in [compatibility-1.0.md](compatibility-1.0.md) and the user guides.
2. Complete every gate and attach evidence to the release candidate commit.
3. Review changelog, known limitations, module API changes, configuration migration, and release archives.
4. Prepare the `v1.0.0` tag and verify the Go module proxy can resolve it.
5. Present the exact tag, artifacts, checksums, and notes for final approval before publishing.

No release tag or publication is authorized by this preparation checklist.

### Named-function template-return follow-up

The 2026-10-09 slice adds `analyse.ResolvedParam.CallableReturnType` metadata
and uses direct/callable argument bindings for named generic function returns.
Go 1.23.12/macOS arm64 full tests and vet plus focused analysis/command/rule
race tests pass. The fresh PSL reference comparison and unchanged Symfony/
Composer issue sets are recorded in [reviewed corpus quality](reviewed-corpus-quality.md).
The diagnostic-quality and performance gates remain open.

The host's Go 1.27.1 full suite failed before this change: parallel
`testing.AllocsPerRun` panics in `TestWithRelativeClassNamesUnchangedDoesNotAllocate`,
and `TestSplitLinesCachedAndDelete` fails pointer-based cache reuse. These
failures require a separate compatibility fix; the minimum-toolchain result
does not establish unrestricted newer-Go compatibility.
