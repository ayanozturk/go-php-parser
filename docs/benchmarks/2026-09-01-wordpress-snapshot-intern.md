# WordPress snapshot generation and identifier intern, 2026-09-01

## Outcome

Snapshot construction now inserts narrowing facts directly, reuses stored control-flow reachability, keeps one- and two-successor CFG edges inline, and interns mixed-case ASCII identifier folds. Identifier lookups that were already lowercase still return the original string. Public fact, graph, and diagnostic contracts are unchanged.

This batch is accepted as an allocation improvement on top of the uncommitted method-resolver views, and the interleaved cold comparison against last commit `9a4f4a4` passes the 5% CV contract. File and diagnostic accounting was identical throughout. This is not a Mago comparison.

## Reproducible source and workload

- Baseline engine commit: `9a4f4a448c22944f8772c96cc76a11d4b3581db8`.
- Candidate: uncommitted method-resolver views plus this snapshot/CFG/identifier-intern batch, applied to `9a4f4a4`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Build: `go build -trimpath -ldflags='-s -w' ./cmd/benchmark` for both engines.
- Protocol: one validation and one process-cold warmup per engine, ten alternating-order measured rounds plus four extra measured runs, 250ms settle pause, 5% maximum CV.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.27.0.

Generated JSON and profiles remain local artifacts and are not committed.

## Allocation profile

Both profiles used three in-process full-analysis iterations. Every iteration parsed 5,357/5,357 files and emitted exactly 22,387 diagnostics.

| Engine | Allocated space | Decision |
| --- | ---: | --- |
| Method-view working tree (pre-intern) | 4.49 GB | Profile reference for this slice |
| Candidate | 3.67 GB | 18.3% lower |

The previous approximately 0.87 GB `asciiLowerIdent` site left the hot list after intern. Remaining leaders are semantic-fact insertion (0.77 GB), `functionScope.clone` (0.38 GB), and control-flow scope/reachability maps.

## Accepted process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.392s | 2.354s | 2.257s | 2.662s | 4.97% | 1,205,895,168 bytes | Accepted |
| Baseline `9a4f4a4` | 2.601s | 2.568s | 2.508s | 2.765s | 3.04% | 1,321,484,288 bytes | Accepted |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 22,387 diagnostics. Both full-sample CVs pass the 5% contract after four extra measured rounds; no outlier was removed.

This supports an 8.0% mean, 8.3% median, and 8.7% maximum-RSS improvement against last commit. That comparison includes the uncommitted method-resolver views as well as this slice.

## Decision

Retain direct fact insertion, compact CFG successor storage, shared statement-list ownership for variable-flow construction, ASCII folding on remaining identifier paths, and the bounded identifier intern. Continue from the new profile leaders: semantic-fact insertion, remaining per-clone scope object cost, and leftover control-flow reachability maps. Isolated-host Mago comparison remains a later gate.
