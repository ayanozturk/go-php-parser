# WordPress duplicate method-parameter copy removal, 2026-09-01

## Outcome

`SemanticSnapshot.ResolveMethod` and `ResolveOwnMethod` no longer clone resolved parameter slices a second time. Their `ProjectIndex` delegates already return fresh defensive copies, so the snapshot layer now preserves the same public mutation-isolation contract with one copy instead of two.

This batch is accepted as an allocation improvement against exact baseline `cbb469b`. Three-pass allocated space fell by 3.5%, and the approximately 0.21 GB snapshot-level `ResolveMethod` allocation site disappeared. The cold comparison is rejected because baseline CV exceeded the 5% contract; no timing or RSS claim follows from it.

## Reproducible source and workload

- Baseline engine commit: `cbb469ba4dfaadd83a85bf0f86b4182388935122`.
- Candidate: the duplicate method-parameter copy removal applied to `cbb469b`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Build: `go build -trimpath -ldflags='-s -w' ./cmd/benchmark` for both engines.
- Protocol: one validation and one process-cold warmup per engine, ten alternating-order measured rounds, 250ms settle pause, 5% maximum CV.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.26.2.

Generated JSON and profiles remain local artifacts and are not committed.

## Allocation profile

Both profiles used three in-process full-analysis iterations. Every iteration parsed 5,357/5,357 files and emitted exactly 26,321 diagnostics.

| Engine | Allocated space | Decision |
| --- | ---: | --- |
| Baseline `cbb469b` | 5.11 GB | Profile reference |
| Candidate | 4.93 GB | 3.5% lower |

The prior approximately 0.21 GB `SemanticSnapshot.ResolveMethod` flat allocation site disappeared. `ProjectIndex.resolveMethodWithTemplates` remains in the profile because it owns the required defensive copy and generic binding behavior. Focused tests prove repeated snapshot method and own-method results remain mutation-isolated and that the snapshot facade adds no allocation beyond its project-index delegate.

## Rejected process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.703s | 2.678s | 2.566s | 2.855s | 3.37% | 1,306,099,712 bytes | Rejected with baseline |
| Baseline `cbb469b` | 2.797s | 2.773s | 2.598s | 3.124s | 5.33% | 1,282,916,352 bytes | Rejected |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 26,321 diagnostics. Baseline CV exceeded the 5% contract, so the raw 3.3% mean direction and RSS samples are not accepted performance evidence.

## Decision

Retain the duplicate-copy removal because it eliminates a measured allocation source while keeping the existing public defensive boundary. Do not describe this batch as a cold speed or memory improvement. Continue from the new profile leaders: semantic-fact insertion, ASCII identifier folding, function-scope metadata copying, control-flow graph storage, and required project-index method resolution.
