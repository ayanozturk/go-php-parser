# Ternary narrowing resource guardrail, 2026-09-05

The ternary correctness iteration reuses branch-local scopes in the existing expression and hover traversals. It adds no whole-project pass. All 290 differential fixtures match PHPStan 2.2.5; parser tests, race tests, vet, and builds pass.

## Workload and protocol

- Exact baseline: `264a964` (the parent of this iteration).
- WordPress: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src,tests,vendor`, excluding `src/js`; all registered rules; eight workers and `GOMAXPROCS=8`.
- Host: Apple M1, macOS ARM64, Go 1.27.0.
- Both binaries built with `go build ./cmd/benchmark` from the candidate and an archive of the exact baseline.
- One validation and one unmeasured cold warmup per engine; ten alternating-order measured cold rounds, 250 ms settle pauses, no extra rounds, 5% CV gate.

Every full-analysis run accounts for 5,357/5,357 files, 1,451,208 LOC, 47,344,277 bytes, and zero failures. Baseline diagnostics are stable at 56,355; candidate diagnostics are stable at 56,316. The 39-report reduction is an audit indicator, not independently established whole-corpus diagnostic parity.

## Resource observations

| Engine | Cold mean | Cold median | CV | Maximum peak RSS |
| --- | ---: | ---: | ---: | ---: |
| Baseline | 2,300.2 ms | 2,255.5 ms | 5.04% | 1,029.6 MB |
| Candidate | 2,325.7 ms | 2,292.0 ms | 3.42% | 1,057.0 MB |

The comparison is **rejected** because baseline CV exceeds 5%. No accepted timing improvement, regression, or Mago resource-envelope claim follows from these numbers. Dropping the slowest run is not allowed. This is a previous-engine guardrail, not a fresh Mago comparison.

Separate three-pass heap profiles retain identical per-engine file and diagnostic accounting. Sampled allocated space is 3,221.65 MB for baseline and 3,338.55 MB for candidate, about 3.6% higher. These sampled totals are a storage-cost indicator, not an accepted allocation optimization. Generated reports, logs, profiles, and the baseline archive remain local under `/tmp/ternary-*` and `/tmp/phpstrom-ternary-baseline`.
