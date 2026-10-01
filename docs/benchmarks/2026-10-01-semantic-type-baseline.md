# Semantic and type baseline, 2026-10-01

## Scope and outcome

This baseline records focused type-system checks and full-registry results for
the four pinned Symfony, WordPress, PSL, and Magento corpora. Ten-run cold
measurements pass the variance gate on all four. The benchmark classifies the
single known malformed Symfony fixture by exact relative path; all other
parser errors and every read failure remain rejection conditions. WordPress
now completes after bounding optional guaranteed-loop-exit flow refinement.
The active plan's four-corpus baseline gate is complete.

No Mago binary was available on this host. These are single-engine baselines,
not comparative performance claims.

## Reproduction inputs

- Parser checkout tested: `5b7b1ec3` (including the loop-flow and accounting
  changes in this report).
- Corpus revisions, pinned by `test_projects/manifest.json`:
  - Symfony: `ae256f91a9cacc470fe77eca87aedd81c65ca55e`.
  - WordPress: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
  - PSL: `331f3ab363508825e62c42725f96c091bcceb970`.
  - Magento: `755e34dd689021c5165db9d35ecff74f7dc51527`.
- Host: AMD Ryzen 7 3700X, 16 logical CPUs, 32 GiB RAM, Linux x86_64.
- Go: `go1.27.1`; benchmark worker count and `GOMAXPROCS`: 4.
- Cold protocol: 1 unmeasured validation, 1 cold warmup, 10 measured process-cold
  full-analysis runs, 250 ms settling delay, no extra runs, warm loop skipped.
- Benchmark command for each root:
  `go run ./cmd/benchmark --root test_projects/<name> --cold-runs 10 --extra-cold-runs 0 --skip-warm --workers 4 --json --output /tmp/go-php-parser-<name>-benchmark-current.json`.
- Symfony's accepted command adds
  `--expected-parse-errors src/Symfony/Component/Config/Tests/Fixtures/ParseError.php`.
  The expected-path list is matched exactly against parser-error paths; any
  read error, missing expected error, or additional parser error rejects the
  run. Per-run paths are included in JSON.
- Snapshot command for each root:
  `go run ./cmd/analysis-corpus-snapshot --root test_projects/<name> --workers 4 --output /tmp/go-php-parser-<name>-current.json`.
- Focused checks:
  `GOWORK=off go test ./...`;
  engine-only differential packs for levels 3, 5, and 7;
  `GOWORK=off go test ./analyse -run '^$' -bench 'Benchmark(ParseTypeCacheHit|ParseTypeCacheMiss|SemanticSnapshotConstructionAllocation|BuildProjectIndexIncremental/full)$' -benchmem -count=5`.

Generated JSON and heap profiles are local `/tmp` artifacts and are not
committed. The fetched corpora are pinned and ignored by Git.

## Full-registry issue snapshots

| Corpus | Semantic targets | Files with issue entries | Parse diagnostics | Issue entries |
| --- | ---: | ---: | ---: | ---: |
| Symfony | 10,026 | 10,026 | 1 | 103,912 |
| PSL | 3,319 | 3,319 | 17 | 35,436 |
| Magento | 25,389 | 25,389 | 1 | 281,641 |
| WordPress | 3,187 | 3,187 | 0 | 78,687 |

The snapshot runner excludes files with parser diagnostics from the indexed
project, records those paths in `parseErrors`, and records an issue set for
every successfully indexed target. WordPress has 3,188 discovered PHP paths in
the benchmark; one path under `src/js/_enqueues/vendor/` is excluded from
semantic targets as vendored code, leaving 3,187 snapshot targets. The
benchmark still parses and accounts for all 3,188 files. The Symfony snapshot
has one parser diagnostic for the malformed fixture listed below. The
benchmark's diagnostic totals are not directly comparable with snapshot issue
entry counts.

## Process-cold measurements

All benchmark runs used the full discovered PHP path (`.`), all registered
rules, four workers, and ten measured cold runs. The strict validation result
is shown separately from the raw timing sample.

| Corpus | Files discovered / parsed / failed | LOC | Bytes | Diagnostics per run | Mean | CV | Max RSS | Harness validation |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| PSL | 3,336 / 3,336 / 0 | 316,742 | 9,436,242 | 36,153 | 1.695 s | 0.99% | 164.2 MiB | Pass |
| Symfony | 10,028 / 10,027 / 1 | 1,847,951 | 80,347,084 | 103,912 | 10.495 s | 2.32% | 724.7 MiB | Pass: expected parse error |
| Magento | 25,390 / 25,390 / 0 | 3,174,100 | 97,609,952 | 281,667 | 14.841 s | 0.72% | 1,044.3 MiB | Pass |
| WordPress | 3,188 / 3,188 / 0 | 1,115,826 | 36,103,588 | 78,687 | 4.271 s | 0.66% | 218.9 MiB | Pass |

All four accepted runs passed exact file accounting and the 5% CV threshold.
Symfony's one failed parse is named exactly as an expected malformed test
fixture; the run has no read failures or other parser-error paths. No Mago
comparison was run.

## WordPress memory finding

The first full-registry snapshot attempt reached about 24.3 GB RSS and was
killed by the OS (exit 137). A heap profile isolated the growth to
`simulateLoopFlowStatements` while refining guaranteed `while`-loop exits: a
branch-heavy loop in WordPress's post-list table produced exponentially many
cloned flow scopes. A 128-path budget now bounds both AST and CST simulations;
on overflow, the optional refinement keeps the incoming scope. Six independent
branches remain below the cap and preserve precision; eight exceed it and
retain the conservative nullable result.

After the fix, the full pinned WordPress snapshot completed with 3,187 semantic
targets and no parser errors. A full-index 25-file analysis probe completed in
about 32 seconds with about 2.7 MB live Go heap after GC; the previously heavy
post-list-table target completed in about 10 seconds with about 2.1 MB live
heap after GC. The ten-run full benchmark peaked at 229,580,800 bytes RSS
(218.9 MiB) and reported no failed files. Issue-set comparisons for PSL,
Symfony, and Magento against their pre-change snapshots had zero mismatches.

## Focused type checks and microbenchmarks

- `GOWORK=off go test ./...`: passed.
- Level 3, 5, and 7 engine-only differential packs: 48/48, 64/64, and 14/14
  fixture expectations passed respectively.
- Five-run Go microbenchmarks, median observations:
  - `BenchmarkSemanticSnapshotConstructionAllocation`: 73.6 µs/op,
    13,944 B/op, 123 allocs/op.
  - `BenchmarkParseTypeCacheHit`: 34.7 ns/op, 0 B/op, 0 allocs/op.
  - `BenchmarkParseTypeCacheMiss`: 5.8 µs/op, 1,330 B/op, 29 allocs/op.

These focused measurements are local engineering baselines, not cross-machine
targets. The filtered benchmark expression did not select the nested
`BuildProjectIndexIncremental/full` benchmark, so no result for that benchmark
is claimed.

## Next work

The active plan moves to template bounds, variance, and generic
inheritance/substitution. Keep subsequent type-system slices gated by focused
clean, mismatch, and boundary fixtures plus the relevant level 3/5/7
differential suites. Preserve the 128-path loop-flow budget as a conservative
fallback for optional precision refinement.
