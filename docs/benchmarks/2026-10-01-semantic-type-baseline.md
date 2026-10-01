# Semantic and type baseline, 2026-10-01

## Scope and outcome

This baseline records focused type-system checks and fresh full-registry results
for the pinned Symfony, PSL, and Magento corpora. Ten-run cold measurements
passed the harness variance and accounting gates for PSL and Magento. Symfony
has a stable cold sample, but its validation run failed to parse one file, so
the harness correctly rejected it. WordPress could not complete either a full
registry snapshot or a cold benchmark within a safe memory envelope on this
desktop. The active plan's four-corpus baseline gate remains open.

No Mago binary was available on this host. These are single-engine baselines,
not comparative performance claims.

## Reproduction inputs

- Parser checkout: `11cc3f9fd61d3c2b5cd945550d8d75a6f45ff05a`.
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
- Snapshot command for each completed root:
  `go run ./cmd/analysis-corpus-snapshot --root test_projects/<name> --workers 4 --output /tmp/go-php-parser-<name>-current.json`.
- Focused checks:
  `GOWORK=off go test ./...`;
  engine-only differential packs for levels 3, 5, and 7;
  `GOWORK=off go test ./analyse -run '^$' -bench 'Benchmark(ParseTypeCacheHit|ParseTypeCacheMiss|SemanticSnapshotConstructionAllocation|BuildProjectIndexIncremental/full)$' -benchmem -count=5`.

Generated JSON and heap profiles are local `/tmp` artifacts and are not
committed. The fetched corpora are pinned and ignored by Git.

## Full-registry issue snapshots

| Corpus | PHP files visited | Files with issue entries | Parse diagnostics | Issue entries |
| --- | ---: | ---: | ---: | ---: |
| Symfony | 10,026 | 10,026 | 1 | 103,912 |
| PSL | 3,319 | 3,319 | 17 | 35,436 |
| Magento | 25,389 | 25,389 | 1 | 281,641 |
| WordPress | incomplete | — | — | — |

The snapshot runner excludes files with parser diagnostics from the indexed
project, records those paths in `parseErrors`, and still records an empty or
non-empty issue set for every successfully indexed target. The benchmark
harness has different file-accounting semantics; do not compare its diagnostic
totals directly with snapshot issue-entry counts.

## Process-cold measurements

All benchmark runs used the full discovered PHP path (`.`), all registered
rules, four workers, and ten measured cold runs. The harness's strict validation
result is shown separately from the raw timing sample.

| Corpus | Files discovered / parsed / failed | LOC | Bytes | Diagnostics per run | Mean | CV | Max RSS | Harness validation |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| PSL | 3,336 / 3,336 / 0 | 316,742 | 9,436,242 | 36,153 | 1.695 s | 0.99% | 164.2 MiB | Pass |
| Symfony | 10,028 / 10,027 / 1 | 1,847,951 | 80,347,084 | 103,912 | 8.118 s | 0.40% | 1,027.9 MiB | Reject: one failed file |
| Magento | 25,390 / 25,390 / 0 | 3,174,100 | 97,609,952 | 281,667 | 14.841 s | 0.72% | 1,044.3 MiB | Pass |
| WordPress | incomplete | — | — | — | — | — | at least 17.1 GiB | Stopped for memory safety |

The Symfony CV is below 5%, but its one failed file means the sample is not an
accepted full-workload baseline. PSL and Magento pass the harness's file
accounting and CV checks. No Mago comparison was run.

## WordPress memory finding

The pinned checkout contains 5,362 PHP files. A full-registry snapshot attempt
reached about 24.3 GB RSS and was killed by the OS (exit 137). A second attempt
used `GOMEMLIMIT=4GiB GOGC=50 GOMAXPROCS=4` and two analysis workers; RSS still
grew to about 16.6 GB, so it was interrupted before reaching the previous
failure point. Its post-index heap profile showed about 32 MB of live Go heap,
which places the large memory growth in the per-file analysis phase rather than
the project-index build.

A process-cold benchmark attempt over the full discovered WordPress tree reached
about 17.1 GB RSS during validation and was stopped before it produced a report.
The earlier accepted WordPress/Mago report from 2026-09-01 used a different
source revision and a `src,tests,vendor` workload with `src/js` excluded; the
pinned checkout fetched here has no `vendor` directory. That historical result
does not close the current pinned-corpus baseline.

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

Make WordPress full-registry analysis memory-bounded and identify the per-file
allocation or retention source before attempting another all-corpus run. Then
repair or classify the Symfony failed-file path and rerun the accepted baseline
protocol. Keep type-system work gated by the focused differential and unit
checks above while this corpus baseline remains open.
