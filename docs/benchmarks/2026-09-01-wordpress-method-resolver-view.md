# WordPress allocation-light method resolver view, 2026-09-01

## Outcome

Internal analysis now reads immutable method metadata through private `resolveMethodView` and `resolveOwnMethodView` queries instead of calling the public `ResolveMethod` / `ResolveOwnMethod` APIs, which must defensively clone parameter slices and allocate a heap `seen` map. The public resolver contract, mutation isolation, and generic template rewriting remain unchanged.

This batch is accepted as an allocation improvement against exact baseline `9a4f4a4`. Three-pass allocated space fell by 5.7%, and the previous approximately 0.79 GB public `ResolveMethod` path shrank to 0.23 GB. The interleaved cold comparison is rejected because both CVs exceeded the 5% contract; no timing or RSS claim follows from it.

## Reproducible source and workload

- Baseline engine commit: `9a4f4a448c22944f8772c96cc76a11d4b3581db8`.
- Candidate: the private method-view batch applied to `9a4f4a4`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Build: `go build -trimpath -ldflags='-s -w' ./cmd/benchmark` for both engines.
- Protocol: one validation and one process-cold warmup per engine, ten alternating-order measured rounds plus ten extra measured runs, 250ms settle pause, 5% maximum CV.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.27.0.

Generated JSON and profiles remain local artifacts and are not committed.

## Allocation profile

Both profiles used three in-process full-analysis iterations. Every iteration parsed 5,357/5,357 files and emitted exactly 22,387 diagnostics.

| Engine | Allocated space | Decision |
| --- | ---: | --- |
| Baseline `9a4f4a4` | 4.77 GB | Profile reference |
| Candidate | 4.50 GB | 5.7% lower |

The previous approximately 0.79 GB `ProjectIndex.ResolveMethod` / `resolveMethodWithTemplates` cumulative site fell to 0.23 GB on the remaining type-sensitive path. Internal lookups reuse immutable index-owned parameter storage and lowercase the method name once per query. Focused tests prove public mutation isolation, internal backing-storage reuse, inherited-method resolution, and lower per-call allocation.

## Rejected process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 3.244s | 3.032s | 2.522s | 6.626s | 26.91% | 1,265 MB | Rejected with baseline |
| Baseline `9a4f4a4` | 3.189s | 3.217s | 2.632s | 3.974s | 8.96% | 1,265 MB | Rejected |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 22,387 diagnostics. Both CVs exceeded the 5% contract, including after ten extra measured rounds, so the raw means and RSS samples are not accepted performance evidence.

## Decision

Retain the private borrowed method views because they remove a measured allocation source while keeping the public immutable and generic-binding boundary. Do not describe this batch as a cold speed or memory improvement. Continue from the new profile leaders: semantic-fact insertion, ASCII identifier folding, control-flow graph storage, and the remaining per-clone scope object cost.
