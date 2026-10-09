# Current implementation plan

Status: in progress

Roadmap source: long-term item 7 (full-analyser release), now explicitly selected by the user

Scope: stabilize the parser, analyser, CLI, and PHP Strom integration for a 1.0 release. Do not publish a release until the documented acceptance gates pass.

## Work sequence

1. [x] Make the install/build and command-line documentation match the shipped CLI; add and maintain a 1.0 release checklist.
2. [x] Freeze the supported Go versions, operating systems, architectures, PHP syntax scope, public Go API, configuration schema, and diagnostic/exit-code contracts in [docs/compatibility-1.0.md](docs/compatibility-1.0.md); CI explicitly builds the declared target matrix.
3. [x] Run the correctness gates: full Go tests and vet, race tests, parser lossless and token-parity corpora, differential fixtures, fuzz smoke, and cross-platform builds. Go 1.23.12/Linux amd64 passed tests, vet, race, lexer parity, and all four 2-minute fuzz smokes; Symfony and WordPress identity/accounting were 100%; all six OS/architecture package builds passed locally and in CI at `82b69e45`. Broader analyzer corpus-quality gates remain tracked separately in [docs/releasing-1.0.md](docs/releasing-1.0.md).
4. [ ] Close the reviewed-corpus diagnostic-quality gate. Thresholds, method, pinned fixture results, and a reproducible dependency-aware PSL comparison are published in [docs/reviewed-corpus-quality.md](docs/reviewed-corpus-quality.md); exact fixtures pass, while the PSL corpus gate remains blocked on review/disposition of thousands of differences. Continue triaging level-8 argument and return-type findings after the larger callable-semantics slice below.
5. [ ] Maintain and gradually improve full-analysis and CLI performance for each pinned workload, carrying forward the best accepted result as changes are made.
6. [ ] Review compatibility and migration notes, supported limitations, module versioning, license/dependency notices, reproducible release artifacts, and rollback instructions.
7. [ ] Only after all gates pass, prepare a `v1.0.0` tag and release artifacts for final review. Publication remains a separate explicit action.

## Active evidence and constraints

- Release gates, current blockers, and commands: [docs/releasing-1.0.md](docs/releasing-1.0.md).
- Performance is continuous release-quality work: preserve the current best accepted full-analysis and CLI behavior, and improve it incrementally with each change.
- Existing CI runs `go test ./...`, `go vet ./...`, and short parser/PHPDoc fuzz jobs on Linux. It does not yet establish cross-platform execution or the full corpus release gates.
- Tusk is the only user-facing binary and config name: `make build` emits `tusk`/`tusk.exe`, and automatic config discovery accepts only `tusk.yaml`. The README, source-run examples, command names, output source, and integration comments now use Tusk consistently. `go test ./...`, `go vet ./...`, native/Windows builds, and representative CLI flows pass.
- Preserve lossless parsing, deterministic diagnostics, file accounting, public compatibility, and existing rule semantics. Every behavior fix needs a regression case and current rule documentation where applicable.

## Current diagnostic-quality work

- [x] Bind named-function direct templates and supported callable returns.
- [x] Normalize callable/Closure contracts, preserve callback metadata through
  inheritance and caching, and check known signature variance/arity at level 5.
- [x] Infer unannotated arrows and simple closures with isolated parameter/capture
  scopes and bounded nesting; preserve method-result templates across initializer
  callbacks and honor supported return assertions after CST lowering.
- [x] Fix the reproduced Go 1.27.1 allocation-test and split-lines-cache failures.
- [x] Validate the larger callable slice with exact fixtures, matched PSL/Symfony/
  Composer snapshots, lossless corpora, and resource measurements.
- [ ] Deliver the parser commit through the production Strom pin and verify pinned
  and sibling-development server tests/vet.
- [ ] Continue reviewed-corpus triage: remaining engine-only PSL argument/return
  families contain 356 and 109 findings. Structured parameter binding, dependent
  bounds, complex callable bodies, and shared integer-range normalization remain
  partial. Review the newly exposed Symfony findings before declaring that gate passed.
- [ ] Establish stable accepted resource baselines for every release workload;
  the larger callable measurement is recorded separately from correctness evidence.

Current acceptance evidence and remaining diagnostic dispositions are in
[reviewed corpus quality](docs/reviewed-corpus-quality.md); the matched resource
protocol is in [callable slice measurements](docs/benchmarks/2026-10-09-callable-semantics.md).
The 1.0 stabilization plan remains active. This completed slice does not authorize
release publication or promotion of a separate roadmap item.
