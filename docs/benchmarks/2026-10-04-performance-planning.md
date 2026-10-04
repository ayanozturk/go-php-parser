# Performance planning evidence — 2026-10-04

Purpose: identify current bottlenecks and ground the [2× execution plan](../plan-performance-2x.md).
This is an unchanged-engine baseline and profile record, not an optimization result.

## Environment and workload

Engine: `42f88abb6a1005732901c10d975ed68f12cdea7f`, built without explicit optimization
flags using `/workspace/.tools/go/bin/go` (Go 1.23.12, linux/amd64).
AMD EPYC 9V74 host, three visible logical CPUs, **two CPUs of cgroup quota**
(`200000 100000`), 8 GiB memory limit. Workers and GOMAXPROCS were both 4;
GOGC/GOMEMLIMIT/GOFLAGS were unset. No concurrent benchmark jobs were run.

Both corpora were fetched at their manifest pins, with all `.` paths, no
exclusions, every registered analysis rule, and unchanged vendor policy.
[Provenance JSON](2026-10-04-performance-planning/provenance.json) records corpus
commits, file-content manifest hashes, binary hash, and profile hashes.

| Corpus | Files parsed / discovered | PHP bytes | Diagnostics per run | Mean cold | Median cold | CV | Maximum measured cold RSS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| PSL | 3,336 / 3,336 | 9,436,242 | 37,883 | 4.3307 s | 4.3225 s | 2.93% | 160.40 MiB |
| Composer | 565 / 565 | 4,175,432 | 8,347 | 1.3506 s | 1.3475 s | 4.24% | 89.97 MiB |

There were zero parse failures. Each report contains ten measured process-cold
runs after validation and one unmeasured cold warmup, with 250 ms settle time
and no extra-run extension. Both passed the harness's ≤5% CV/accounting gate.
Raw reports: [PSL](2026-10-04-performance-planning/psl-cold.json),
[Composer](2026-10-04-performance-planning/composer-cold.json).
RSS above is the maximum of measured cold samples, not the separate validation
run. The reports also contain index-only measurements.

Fresh measurements do **not** cover Symfony, WordPress, or Magento. The
[October 1 baseline](2026-10-01-semantic-type-baseline.md) covers those corpora
on different hardware, Go 1.27.1, and an older engine; its absolute timings are
not comparable. These baseline-only runs are not interleaved candidate A/B
experiments and do not establish exact issue-set parity between revisions.

## Profile observations

CPU and heap profiles used three in-process iterations, separate from cold
measurements. PSL sampled 24.82 CPU-seconds over 12.71 s; Composer sampled
7.78 CPU-seconds over 4.14 s. Approximately two cores were active despite four
workers. Allocation profiles report **total sampled allocated bytes**, not live
heap or RSS: about 6.05 GB for PSL and 1.93 GB for Composer across three iterations.

| Cumulative CPU stack | PSL | Composer |
| --- | ---: | ---: |
| `RunAnalysisRulesWithContext` | 34.73% | 36.76% |
| `buildFileSemantics` | 19.06% | 20.82% |
| `generateInferredTypeFactsForFile` | 13.62% | 16.97% |
| `ParseWithContext` | 20.91% | 21.08% |
| `lowerFromResult` | 5.52% | 6.68% |
| `asciiLowerIdent` | 14.50% | 5.27% |
| `boundedStringCache.store` | 7.98% | 0.51% |
| `boundedStringCache.load` | 6.24% | 4.63% |
| `sync.Map.dirtyLocked` | 8.02% | 1.03% |
| `runtime.scanobject` | 18.25% | 13.88% |
| `BuildProjectIndex` | 2.26% | 1.54% |

These are overlapping CPU stacks, **not additive wall-time fractions**. Cache
store's PSL listing attributes 1.92 of its 1.98 cumulative seconds to
`c.values.Store`; verify on the intended compiler because Go's map internals vary.

| Flat allocation site | PSL | Composer |
| --- | ---: | ---: |
| `LexAllContext` | 16.59% | 16.06% |
| `Interner.Token` | 7.40% | 7.24% |
| `Interner.Node` | 5.84% | 6.24% |
| `collectLeadingTrivia` | 5.08% | 5.95% |
| `Type.withRelativeClassNames` | 3.60% | 3.22% |
| `semanticFactStore.putGeneratedInferred` | 2.50% | 3.44% |

The first four sites sum to 34.90% / 35.49%. `ParseWithContext` cumulative
allocation is 45.17% / 44.29%, including descendants; do not add it to that sum.

Retained extracts:

- PSL: [CPU](2026-10-04-performance-planning/psl-cpu-cumulative.txt), [allocations](2026-10-04-performance-planning/psl-alloc-space.txt), [profile log](2026-10-04-performance-planning/psl-profile.txt).
- Composer: [CPU](2026-10-04-performance-planning/composer-cpu-cumulative.txt), [allocations](2026-10-04-performance-planning/composer-alloc-space.txt), [profile log](2026-10-04-performance-planning/composer-profile.txt).

Raw binary profiles were generated in temporary workspace storage; they are
not committed. Extracts and their source profile hashes are retained here.
Printed profile iteration wall times have unequal boundaries (first iteration's
timer excludes initial parsing, later iterations include it); use only unprofiled
cold JSON for wall-time claims.

## Source inspection that determines priority

- `analyse/ascii.go`: bounded cache combines `sync.Map` with locked FIFO eviction;
  the already-lowercase path is already cheap. Test bounded alternatives.
- `analyse/types.go`: `withRelativeClassNames` allocates before recognizing an
  unchanged result. This is a narrow first allocation fix.
- `token/token.go` and `lexer/lexer.go`: trivia uses full token values; full
  parsing retains the `LexAllContext` array. Compact storage has broad potential.
- `analyse/semantic_snapshot.go`: eager flow/inferred-fact construction and
  repeated per-expression context lookup motivate shared function work records.
  Existing lowering caches and fused walks must be reused.
- `syntax/parse_ast.go` and both analysis drivers: declaration indexing precedes
  full host parsing. Index AST references are deliberately dropped; retaining
  all trees may worsen RSS. Measure direct declaration summaries instead.
- `syntax/green.go`: composite hash interning already exists. The
  [October 2 Magento record](2026-10-02-magento-green-node-hash.md) measured only
  a 1.7% improvement from that change; it cannot justify a new 2× claim.

## Reproduction

Use the recorded engine, corpus pins, environment, and compiler. Commands below
use `go` for the recorded toolchain and run from the repository root:

```sh
go run ./cmd/fetch-test-projects --only psl,composer-src
mkdir -p /tmp/php-performance-plan
go build -o /tmp/php-performance-plan/benchmark ./cmd/benchmark
GOMAXPROCS=4 /tmp/php-performance-plan/benchmark \
  --root test_projects/psl --workers 4 --cold-runs 10 --extra-cold-runs 0 \
  --skip-warm --json --output /tmp/php-performance-plan/psl-cold.json
GOMAXPROCS=4 /tmp/php-performance-plan/benchmark \
  --root test_projects/psl --workers 4 --profile-iterations 3 \
  --cpuprofile /tmp/php-performance-plan/psl.cpu \
  --memprofile /tmp/php-performance-plan/psl.heap
go tool pprof -top -cum -nodecount=60 /tmp/php-performance-plan/benchmark /tmp/php-performance-plan/psl.cpu
go tool pprof -top -alloc_space -nodecount=60 /tmp/php-performance-plan/benchmark /tmp/php-performance-plan/psl.heap
```

Repeat sequentially with root `test_projects/composer-src` and output prefix
`composer`. Process-cold includes startup, discovery/read, both parse tiers,
indexing, analysis and worker JSON transport. It excludes the production CLI's
full checksum/cache persistence/report rendering path. The harness warm loop
also excludes its preparatory parse and is not a persistent incremental replay.
The execution plan adds independent CLI and incremental acceptance boundaries.
