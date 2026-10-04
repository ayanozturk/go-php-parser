# Preparing a 1.0 release

This document tracks stabilization for the parser, analyzer, CLI, and PHP Strom integration as one supported system. A successful Linux build or passing unit suite alone is not a 1.0 release signal. Do not publish a tag or binaries until the required checks below have evidence attached to the release candidate.

## Release gates

| Gate | Required evidence | Current state |
| --- | --- | --- |
| Build and CLI contract | `make build` creates `tusk` (`tusk.exe` on Windows); documented commands, current-directory config discovery, output formats, and exit codes match command tests | Passed locally: full tests and vet; native and Windows executable builds; smoke runs for init, style, analyze, AST, tokens, config, file listing, style-rule listing, source invocation, and Make invocation |
| Go API and module contract | Supported Go toolchain range, public packages/symbols, compatibility policy, module path, and dependency licenses reviewed | Open |
| Platform support | CI builds and tests on every declared OS/architecture; release archives are reproducible and contain the expected executable and license | In progress: CI build matrix added for Ubuntu, macOS, and Windows; local Windows amd64 and macOS amd64 cross-builds passed; native full test execution and release archive reproducibility remain open |
| Parser correctness | Lossless identity, gold trees, Zend token parity, corpus identity/accounting, and malformed-input fuzz gates pass | Open; full Go tests and race suite pass locally; corpus identity/accounting and scheduled fuzz evidence remain open. See `AGENTS.md` and syntax CI/fuzz jobs |
| Analyzer correctness | Full tests, race suite, exact differential fixtures, reviewed-corpus quality thresholds, and complete corpus accounting pass | Open; full Go tests, race suite, and vet pass locally; reviewed-corpus quality and complete corpus accounting remain open. See [analyser capability matrix](analyser-capability-matrix.md) and roadmap release gate |
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

The CLI's built-in `--help` output lists Go flags; command names and descriptions are verified by `go test ./command`. Smoke-test config initialization in a temporary working directory (`tusk init` must create `tusk.yaml`) and run representative style, AST, token, config, file-list, and analysis commands against an explicit `tusk.yaml` with `-config` before the command.

The fuzz commands are scheduled smoke checks, not exhaustive proof. Also run the corpus identity, differential, PHPStan comparison, PHP Strom integration, cross-platform, and performance workloads required by the rows above. Record exact corpus commits, configuration, Go version, run counts, file/byte totals, parse failures, diagnostics, peak RSS, coefficient of variation, and artifact hashes. Do not turn a missing or unsupported workload into a passing result.

## Version and publication sequence

1. Freeze the release scope and supported compatibility contract in this document and the user guides.
2. Complete every gate and attach evidence to the release candidate commit.
3. Review changelog, known limitations, module API changes, configuration migration, and release archives.
4. Prepare the `v1.0.0` tag and verify the Go module proxy can resolve it.
5. Present the exact tag, artifacts, checksums, and notes for final approval before publishing.

No release tag or publication is authorized by this preparation checklist.
