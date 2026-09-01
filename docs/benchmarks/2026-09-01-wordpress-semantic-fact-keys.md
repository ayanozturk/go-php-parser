# WordPress semantic-fact key storage, 2026-09-01

## Outcome

The per-file semantic-fact store now partitions built-in fact kinds, packs ordinary source spans into one map word, and stores generated inferred facts without reserving the external-value field that only caller-supplied facts use. Custom namespaced fact kinds, caller-supplied values, duplicate rejection, large offsets, exact lookup, deterministic enumeration, and the public immutable API remain unchanged.

This batch is accepted as an allocation improvement against exact baseline `cfc1c50`. The process-cold comparison is rejected because both engines exceeded the 5% CV contract. This is not a Mago comparison.

## Reproducible source and workload

- Baseline engine commit: `cfc1c503be5b7e8e88d874e3c7c2cd308425e076`.
- Candidate: this semantic-fact key-storage batch applied to `cfc1c50`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Allocation protocol: three in-process full-analysis iterations with `--memprofile`.
- Cold protocol: one validation and one process-cold warmup per engine, ten alternating-order measured rounds plus ten extra measured runs, 250ms settle pause, 5% maximum CV.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.27.0.

Generated JSON and profiles remain local artifacts and are not committed.

## Allocation profile

Both profiles used three in-process full-analysis iterations. Every iteration parsed 5,357/5,357 files and emitted exactly 22,387 diagnostics.

| Engine | Allocated space | Semantic-fact insertion | Decision |
| --- | ---: | ---: | --- |
| Baseline `cfc1c50` | 3,873,268,076 bytes | 783,376,032 bytes | Exact committed baseline |
| Candidate | 3,516,256,187 bytes | 435,973,845 bytes | 9.2% / 44.3% lower |

The focused 256-fact regression benchmark records 43,720 bytes per generated built-in insertion batch versus 92,472 bytes for the former filename-partitioned span-plus-kind map. It also gates duplicate behavior, same-span different-kind coexistence, custom kinds, caller values, and the large-offset fallback.

## Rejected process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.924s | 2.915s | 2.459s | 3.487s | 9.00% | 1,074,446,336 bytes | Rejected |
| Baseline `cfc1c50` | 3.378s | 3.136s | 2.624s | 5.419s | 23.44% | 1,152,614,400 bytes | Rejected |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 22,387 diagnostics. The raw timing and RSS direction is not accepted performance evidence because neither engine passed the stability gate.

## Decision

Retain the compact built-in keys and lean generated-inference values because they remove a measured allocation source without narrowing the public fact contract. The next profile leaders are the per-clone scope object and control-flow scope/reachability maps. An isolated-host exact-baseline run remains necessary before making a cold-performance claim.
