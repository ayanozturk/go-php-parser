# Current implementation plan

Status: in progress

Roadmap source: long-term item 7 (full-analyser release), now explicitly selected by the user

Scope: stabilize the parser, analyser, CLI, and PHP Strom integration for a 1.0 release. Do not publish a release until the documented acceptance gates pass.

## Work sequence

1. [x] Make the install/build and command-line documentation match the shipped CLI; add and maintain a 1.0 release checklist.
2. [x] Freeze the supported Go versions, operating systems, architectures, PHP syntax scope, public Go API, configuration schema, and diagnostic/exit-code contracts in [docs/compatibility-1.0.md](docs/compatibility-1.0.md); CI explicitly builds the declared target matrix.
3. [ ] Run the correctness gates: full Go tests and vet, race tests, parser lossless and token-parity corpora, differential fixtures, fuzz smoke, and cross-platform builds. Fix release-blocking failures with focused regression coverage.
4. [ ] Complete the reviewed-corpus compatibility and diagnostic-quality thresholds; publish the methods, exact corpus revisions, and results.
5. [ ] Establish best accepted full-analysis and CLI results for each pinned workload. For every change, run at least ten process-cold repetitions against the same best build with identical workload, toolchain, worker settings, and environment; require CV ≤5%, complete file/diagnostic parity, and no time or peak-memory regression beyond measurement noise. Keep the existing best result unless a repeatable improvement is measured.
6. [ ] Review compatibility and migration notes, supported limitations, module versioning, license/dependency notices, reproducible release artifacts, and rollback instructions.
7. [ ] Only after all gates pass, prepare a `v1.0.0` tag and release artifacts for final review. Publication remains a separate explicit action.

## Active evidence and constraints

- Release gates, current blockers, and commands: [docs/releasing-1.0.md](docs/releasing-1.0.md).
- Performance is a continuous release-quality gate on our own best accepted results. Each change should preserve or improve the current best. Compare on identical pinned workloads, toolchain, worker settings, and environment; require at least ten process-cold repetitions, CV ≤5%, complete file/diagnostic parity, and no time or peak-memory regression beyond measurement noise. Promote a new best score only when the improvement is repeatable. The active gate is our rolling best, without a fixed speedup multiplier or other-analyser comparison.
- Establish a fresh rolling best for the release workload set before using per-change scores as a gate.
- Existing CI runs `go test ./...`, `go vet ./...`, and short parser/PHPDoc fuzz jobs on Linux. It does not yet establish cross-platform execution or the full corpus release gates.
- Tusk is the only user-facing binary and config name: `make build` emits `tusk`/`tusk.exe`, and automatic config discovery accepts only `tusk.yaml`. The README, source-run examples, command names, output source, and integration comments now use Tusk consistently. `go test ./...`, `go vet ./...`, native/Windows builds, and representative CLI flows pass.
- Preserve lossless parsing, deterministic diagnostics, file accounting, public compatibility, and existing rule semantics. Every behavior fix needs a regression case and current rule documentation where applicable.
- Use only the rolling best-result record for active decisions; do not carry forward scores from superseded plans.
