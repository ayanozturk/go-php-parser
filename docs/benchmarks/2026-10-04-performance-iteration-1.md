# Performance implementation, iteration 1

Engine baseline: 42f88abb6a1005732901c10d975ed68f12cdea7f.
Implementation commits: fa015a4f, d42492b0, da81f345.
Go 1.23.12 linux/amd64, AMD EPYC 9V74, 2 CPU cgroup quota, 8 GiB limit,
workers/GOMAXPROCS 4. Same selected files, rules, and manifest pins per pair.
The benchmark tool alternates candidate and baseline runs. Candidate and baseline
binary hashes are listed in the adjacent provenance JSON.

## Accepted interleaved results to date

| Corpus | Baseline mean | Candidate mean | Speed ratio (paired 95% interval) | CV base / candidate | Peak RSS base → candidate | Diagnostics |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| PSL, cache slice | 4.083 s | 3.715 s | 1.099× (1.075–1.123) | 3.49% / 2.61% | 164.98 → 164.36 MiB | 37,883 / 37,883 |
| PSL, current P2 slice | 3.708 s | 3.627 s | 1.022× (1.004–1.041) | 1.51% / 2.44% | 163.08 → 161.93 MiB | 37,883 / 37,883 |
| Symfony | 19.891 s | 17.751 s | 1.121× (1.099–1.143) | 2.03% / 2.21% | 1,021.17 → 974.03 MiB | 108,848 / 108,848 |
| WordPress | 8.406 s | 8.262 s | 1.018× (1.004–1.031) | 1.25% / 2.60% | 362.25 → 340.78 MiB | 87,121 / 87,121 |
| Magento, 2 workers | 43.217 s | 33.137 s | 1.304× (1.291–1.318) | 1.25% / 1.86% | 1,041.42 → 1,231.05 MiB | 285,737 / 285,737 |

All listed pairs passed the benchmark's 5% CV gate. Magento is a qualified
timing win but failed the peak RSS gate; the other pairs showed lower candidate
peak RSS. Exact issue snapshots match for all files on PSL (3,319 indexed),
Composer (532), WordPress (3,187), Symfony (10,026; one expected parse error),
and Magento (25,389 indexed, same five parse diagnostics).

The PSL cache pair compares the implementation against the pre-cache build. The
PSL P2 pair compares against the cache-only build. Symfony and WordPress compare
the current engine against the frozen original baseline.

## Inconclusive Composer measurement

The 30-run Composer cache comparison produced 1.329 s baseline and 1.338 s
candidate means, but both CVs were above 6% (5% required). It is not accepted
as either a gain or a regression. Candidate peak RSS was lower (86.75 MiB vs
88.04 MiB). Keep the raw run and collect a qualified same-host measurement
before making a Composer timing claim.

## Changes implemented

- analyse/ascii.go: replace one global sync.Map plus FIFO store lock with an
  8-shard map guarded by per-shard read/write locks. Each shard has a bounded
  FIFO and limits; aggregate limits are 8,192 keys and 512 KiB of keys. Maps grow
  with observed keys. PSL and large framework A/B runs support the cache change;
  small-corpus timing needs another qualified sample.
- analyse/types.go: avoid allocating replacement maps when relative class
  types do not occur. The unchanged-path test asserts zero allocations. The
  mapping probe checks candidate validity without constructing throwaway type
  atoms.
- analyse/semantic_snapshot.go: resolve each function's generated-fact subject
  once, outside the per-expression walk.

Profiles still place most allocation in parsing: the current PSL sample assigns
17.94% flat allocation to LexAllContext, 7.61% to Interner.Token, 6.56% to
Interner.Node, and 5.25% to leading trivia. The symbol hoist reduced the
cumulative CPU stack under addGeneratedInferredTypeFact from 9.24% to 7.02%
in separate directional profiles. See the adjacent candidate allocation extract.

## Outstanding gates

- Magento peak RSS must be brought to or below its baseline maximum. The current
  candidate maximum is 1,231.05 MiB vs 1,041.42 MiB baseline despite a lower
  median; all samples, including outliers, remain in the gate.
- The production CLI comparison has not run. These are engine-harness timings,
  not full CLI acceptance.
- The 2× target is not met: qualified full-workload ratios range from 1.018× to
  1.304×. The P1/P2 cumulative milestone (≤0.88× original baseline) is not met
  on the measurements available. Continue from fresh profiles and measure
  production CLI; do not count partial paths or warm-loop savings.

Raw machine-readable records are retained in the adjacent iteration-1 directory.
Heap profiles and binaries were temporary; their hashes/builds are in provenance
and profile extracts are committed.

## Magento timing and memory

At four workers the pinned Magento process-cold baseline had mean 42.608 s but
CV 6.15%, so it failed the 5% gate. The qualified paired comparison used two
workers on both frozen baseline and current candidate to match the two-CPU quota.
It passed timing CV: 43.217 s baseline vs 33.137 s candidate, ratio 1.304×
(paired 95% interval 1.291–1.318), with 285,737 diagnostics in every run.
Files and bytes matched: 25,390 / 25,390 parsed, 97,609,952 PHP bytes.

The peak RSS gate **failed**: maximum measured sample was 1,041.42 MiB baseline
and 1,231.05 MiB candidate. Candidate median RSS was lower (927.60 MiB vs
944.71 MiB), but the two candidate outliers reached 1,115 and 1,231 MiB. These
samples remain included; do not discard them. Magento therefore does not pass
full acceptance until memory behavior is explained and the max RSS gate passes.
The baseline and candidate issue snapshots match across 25,389 indexed files,
including the same parse-error file:
dev/tests/integration/testsuite/Magento/CatalogUrlRewrite/Observer/ProcessUrlRewriteOnChangeVisibilityObserverTest.php.
The benchmark parser accepted all 25,390 files; the snapshot pipeline reports
five parse diagnostics in this same fixture on each engine.

## Rejected compact green-token prototype

A prototype replaced each interned token's full trivia-bearing Token with a
compact green token and materialized public tokens on Token() calls. Syntax and
analyse tests passed, and the 10-run PSL timing comparison passed CV, but it was
0.980× baseline (3.731 s vs 3.658 s; paired 95% interval 0.956–1.006). Peak RSS
was slightly lower.

The allocation profile rejected the design: total sampled allocation rose from
5.74 GB to 6.12 GB over three iterations. Green trivia compaction saved bytes
inside Interner.Token, but publicToken materialization added 549 MB of
allocations and compactGreenToken added 140 MB. The prototype was removed from
the implementation branch; the extracts and raw A/B record remain here to avoid
repeating this design. The next P3 attempt must let internal consumers read
compact token fields without converting them to the public Token form.
