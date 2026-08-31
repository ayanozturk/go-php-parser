# WordPress snapshot allocation batch, 2026-08-31

## Outcome

The snapshot-backed pipeline now stores semantic facts in compact per-file maps, lowercases uppercase ASCII identifiers with one allocation instead of two, and collects method-receiver diagnostics during the existing argument walk. The public immutable fact API and diagnostic output are unchanged.

A three-iteration WordPress allocation profile fell from 8.43 GB at exact baseline `de5598e` to 6.46 GB in the candidate, a 23.3% reduction. The profile is deterministic engineering evidence, not a cold-performance claim.

The exact-baseline cold comparison and the contemporaneous Mago comparison were both rejected by the 5% CV contract. No accepted speed or Mago-parity claim follows from this batch.

## Reproducible source and workload

- Baseline engine commit: `de5598e`.
- Candidate: the five-file allocation batch applied to `de5598e`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Mago: verified 1.47.4 Apple ARM64 release archive, SHA-256 `ecfd09ff7700fe332b5ab95dde08e6dc78d2884de42a58cc36474c50e4a4a2b0`, using `benchmark-configs/mago/wordpress-develop.toml` and eight threads.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.26.2. The host was not isolated.

Generated JSON and profiles remain local artifacts and are not committed.

## Allocation profile

Both profiles ran three in-process full-analysis iterations. Every iteration parsed 5,357/5,357 files and emitted exactly 26,321 diagnostics.

| Engine | Allocated space | Decision |
| --- | ---: | --- |
| Baseline `de5598e` | 8.43 GB | Profile reference |
| Candidate | 6.46 GB | 23.3% lower |

The compact fact store reduced generated-fact insertion from approximately 1.54 GB to 0.75 GB in the sampled profile. `copyTypeMap` is now the largest flat allocation site at approximately 1.04 GB. The remaining profile stays GC-heavy, so further work should target measured scope copying and fact generation rather than disabling diagnostics.

## Exact-baseline process-cold indicator

The candidate harness performed validation and warmup runs, then alternated candidate-first and baseline-first order for ten measured rounds.

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.964s | 2.911s | 2.838s | 3.322s | 5.15% | 1,357,496,320 bytes | Rejected |
| Baseline `de5598e` | 3.435s | 3.449s | 3.172s | 3.720s | 4.12% | 1,665,318,912 bytes | Rejected with candidate |

All validation, warmup, and measured runs retained identical accounting: 5,357/5,357 files, 1,451,208 LOC, 47,344,277 bytes, zero failures, and 26,321 diagnostics. The raw mean was 13.7% lower and maximum peak RSS 18.5% lower, but candidate CV exceeded the contract by 0.15 percentage points. The direction is useful allocation evidence only.

## Contemporaneous Mago indicator

Ten process-cold runs alternated tool order. The engine retained the same file and diagnostic accounting as above. Mago emitted a stable 218,741 findings: 143,188 errors, 72,386 warnings, 3,129 help findings, and 38 notes. Mago's reporting still did not expose comparable analysed-file accounting, and the semantic coverage is materially different.

| Tool | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| go-php-parser candidate | 3.043s | 2.965s | 2.840s | 3.790s | 8.72% | 1,258,995,712 bytes | Rejected |
| Mago 1.47.4 | 3.590s | 3.595s | 3.260s | 4.160s | 6.76% | 1,092,829,184 bytes | Rejected |

The raw engine/Mago ratios were 0.848x mean time and 1.152x maximum peak RSS, inside the numerical Mago-class thresholds. They are not accepted results because both CVs failed, Mago file accounting remains unknown, and enabled semantic coverage is not comparable.

## Decision

Retain all three optimizations. Continue performance work from the new profile, beginning with `copyTypeMap` and remaining snapshot-generation allocation. Rerun the interleaved gates on an isolated host before making a cold-performance or Mago comparison claim.
