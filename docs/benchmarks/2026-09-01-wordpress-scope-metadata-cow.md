# WordPress function-scope metadata copy-on-write, 2026-09-01

## Outcome

Function-scope clones now share array-index and generic-instance metadata until the first real write. Array-index slices and generic type-argument slices are copied when they enter a scope; map detachment is shallow because stored slices are immutable internally. Missing-key clears do not detach shared maps, and empty generic maps are allocated lazily.

This batch passes both the allocation and exact-baseline process-cold gates. Against `1696897`, three-pass allocated space fell by 3.7%, cold mean by 3.8%, median by 4.0%, and maximum peak RSS by 0.3%. File and diagnostic accounting was identical throughout.

This is an accepted improvement against the exact previous engine. It is not a Mago comparison or parity claim.

## Reproducible source and workload

- Baseline engine commit: `16968976b8ecb535fe14021f957c7ad97ebb8d9e`.
- Candidate: the scope-metadata copy-on-write batch applied to `1696897`.
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
| Baseline `1696897` | 4.93 GB | Profile reference |
| Candidate | 4.75 GB | 3.7% lower |

The baseline clone path allocated approximately 0.16 GB in eager array-index copies and 0.09 GB in eager generic-context copies. Generic-context copying disappeared from the candidate hot list; array-index copying fell to approximately 0.06 GB and now represents actual first-write detachments. The remaining `functionScope.clone` flat cost is the scope object itself.

## Accepted process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.667s | 2.628s | 2.564s | 2.824s | 3.17% | 1,290,715,136 bytes | Accepted |
| Baseline `1696897` | 2.773s | 2.738s | 2.647s | 3.070s | 4.08% | 1,295,155,200 bytes | Accepted |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 26,321 diagnostics. Both full-sample CVs pass the 5% contract; no outlier was removed.

## Correctness and ownership

Tests cover shared read-only backing maps; original, child, and sibling write/delete isolation; missing-key no-op clears; caller mutation of input slices; defensive array-index lookup results; and a one-allocation ceiling for read-only clones carrying metadata. All production writes are routed through the copy-on-write helpers.

## Decision

Retain scope-metadata copy-on-write and lazy generic-map allocation. Continue from the new profile leaders: semantic-fact insertion, ASCII identifier folding, required project-index method resolution, control-flow graph storage, and the irreducible-per-clone scope object cost.
