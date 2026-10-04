# Current implementation plan

Status: in progress

Roadmap source: long-term item 7 (full-analyser release), now explicitly selected by the user

Scope: stabilize the parser, analyser, CLI, and PHP Strom integration for a 1.0 release. Do not publish a release until the documented acceptance gates pass.

## Work sequence

1. [ ] Make the install/build and command-line documentation match the shipped CLI; add and maintain a 1.0 release checklist.
2. [ ] Freeze the supported Go versions, operating systems, architectures, PHP syntax scope, public Go API, configuration schema, and diagnostic/exit-code contracts.
3. [ ] Run the correctness gates: full Go tests and vet, race tests, parser lossless and token-parity corpora, differential fixtures, fuzz smoke, and cross-platform builds. Fix release-blocking failures with focused regression coverage.
4. [ ] Complete the reviewed-corpus compatibility and diagnostic-quality thresholds; publish the methods, exact corpus revisions, and results.
5. [ ] Re-run the stable full-analysis and CLI performance gates, including the open 2× self-baseline objective, Composer regression comparison, memory limits, and complete file/diagnostic accounting.
6. [ ] Review compatibility and migration notes, supported limitations, module versioning, license/dependency notices, reproducible release artifacts, and rollback instructions.
7. [ ] Only after all gates pass, prepare a `v1.0.0` tag and release artifacts for final review. Publication remains a separate explicit action.

## Active evidence and constraints

- Release gates, current blockers, and commands: [docs/releasing-1.0.md](docs/releasing-1.0.md).
- Implementation A/B measurements and raw records: [performance iteration 1](docs/benchmarks/2026-10-04-performance-iteration-1.md). Current speed ratios are 1.018–1.304×; the 2× objective is unmet, Magento's peak RSS regressed, and Composer timing did not pass the variance gate.
- Existing CI runs `go test ./...`, `go vet ./...`, and short parser/PHPDoc fuzz jobs on Linux. It does not yet establish cross-platform execution or the full corpus release gates.
- The latest release-surface correction aligns `make build` with the documented `go-phpcs` executable and replaces obsolete CLI examples. Continue checking user-facing commands against the command registry and command tests.
- Preserve lossless parsing, deterministic diagnostics, file accounting, public compatibility, and existing rule semantics. Every behavior fix needs a regression case and current rule documentation where applicable.
- Keep the performance plan and historical results as evidence; they no longer define the active task sequence.
