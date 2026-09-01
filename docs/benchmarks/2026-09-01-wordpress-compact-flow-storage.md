# WordPress compact control-flow storage, 2026-09-01

## Outcome

Control-flow snapshots now partition graphs and reachability by filename, pack ordinary statement/scope spans, encode built-in scope kinds, combine reachable/unreachable/ambiguous state in one map, keep the first graph inline, and index additional graphs into a dense per-file slice. Stored graph blocks retain only local statement offsets; public graph blocks reconstruct the filename and keep defensive successor copies. Custom scope kinds and source offsets above `uint32` retain a general fallback.

This batch passes both the allocation and exact-baseline process-cold gates against `f98411e`. It is not a Mago comparison.

## Reproducible source and workload

- Baseline engine commit: `f98411ef2d8729f0f9cabd50ef92fd9ae63ad2c2`.
- Candidate: this compact control-flow storage batch applied to `f98411e`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Allocation protocol: three in-process full-analysis iterations with `--memprofile`.
- Cold protocol: one validation and one process-cold warmup per engine, ten alternating-order measured rounds, 250ms settle pause, 5% maximum CV.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.27.0.

Generated JSON and profiles remain local artifacts and are not committed.

## Allocation profile

Both profiles used three in-process full-analysis iterations. Every iteration parsed 5,357/5,357 files and emitted exactly 22,387 diagnostics.

| Engine | Allocated space | Graph/reachability sites | Decision |
| --- | ---: | ---: | --- |
| Baseline `f98411e` | 3,273,965,861 bytes | 609,688,347 bytes | Exact committed baseline |
| Candidate | 3,046,140,014 bytes | 303,490,806 bytes | 7.0% / 50.2% lower |

The combined site total includes graph insertion, statement reachability, and linear graph construction in each profile. Adversarial tests cover graph lookup, parent resolution, duplicate-scope first-writer behavior, ambiguous spans, cross-file isolation, defensive block copies, custom kinds, large offsets, and bounded allocation growth across many linear scopes.

## Accepted process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.017s | 2.024s | 1.926s | 2.121s | 3.11% | 1,022,656,512 bytes | Accepted |
| Baseline `f98411e` | 2.141s | 2.147s | 2.050s | 2.223s | 1.97% | 1,075,625,984 bytes | Accepted |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 22,387 diagnostics. Both full-sample CVs pass the 5% contract without extra measured runs or outlier removal.

This supports a 5.8% mean, 5.7% median, and 4.9% maximum-RSS improvement against the exact baseline.

## Decision

Retain the compact per-file flow store and dense graph ownership. The queued profile-led storage reductions are now complete enough to run the contemporaneous same-machine Mago comparison. Further tuning should follow that comparison and the new profile leaders rather than reopening already-cleared sites without evidence.
