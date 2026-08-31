# WordPress allocation-light function resolver view, 2026-08-31

## Outcome

Internal analysis now reads immutable function metadata through a private borrowed view instead of calling the public `SemanticSnapshot.ResolveFunction` API, which must defensively clone parameter slices. The public resolver contract and its mutation isolation remain unchanged.

This batch is accepted as an allocation improvement against exact baseline `0f8eee4`. It reduced three-pass allocated space by 7.8% and removed the approximately 0.44 GB `SemanticSnapshot.ResolveFunction` allocation site from the profile. The stable cold comparison was runtime-neutral, so no speed improvement is claimed.

## Reproducible source and workload

- Baseline engine commit: `0f8eee4255fbbfc29f749a699cc376f83cf4f358`.
- Candidate: the private function-view batch applied to `0f8eee4`.
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
| Baseline `0f8eee4` | 5.54 GB | Profile reference |
| Candidate | 5.11 GB | 7.8% lower |

The previous approximately 0.44 GB `SemanticSnapshot.ResolveFunction` flat allocation site disappeared. The public API still clones parameter slices, while private analyser lookups reuse immutable index-owned storage. Focused tests prove public mutation isolation, internal backing-storage reuse, and lower per-call allocation.

## Stable process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.434s | 2.430s | 2.351s | 2.525s | 2.00% | 1,346,846,720 bytes | Runtime neutral |
| Baseline `0f8eee4` | 2.431s | 2.426s | 2.332s | 2.660s | 3.54% | 1,366,163,456 bytes | Runtime reference |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 26,321 diagnostics. Both CVs pass the 5% stability contract. The 0.1% mean difference and 0.2% median difference are treated as neutral; maximum peak RSS was 1.4% lower.

## Decision

Retain the private borrowed resolver view because it removes a measured allocation source without weakening the public immutable boundary. Do not describe this batch as a cold speed improvement or a Mago comparison. Continue from the new profile leaders: semantic-fact insertion, ASCII identifier folding, method resolver result construction, function-scope metadata copying, and control-flow graph storage.
