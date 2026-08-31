# WordPress persistent scope layers, 2026-08-31

## Outcome

Branch-sensitive function scopes now retain variable and property types in bounded persistent layers. A clone shares its immutable layer chain, the first write uses an inline one-entry delta, and only a second distinct write allocates a local map. Chains compact at a declared depth limit. Callable and array-shape metadata retain their existing copy-on-write representation, with missing-key deletion guards that avoid unnecessary detachment.

This batch passes the exact-baseline process-cold gate. Against `6fdfba0`, the candidate reduced cold mean by 3.1%, median by 4.1%, and maximum peak RSS by 5.4%. Both engines met the 5% CV contract with identical file and diagnostic accounting.

This is an accepted improvement against the exact previous engine. It is not a Mago comparison or parity claim.

## Reproducible source and workload

- Baseline engine commit: `6fdfba0ff52b679f0dc1d50dceab58a2b0f6b065`.
- Candidate: the persistent-scope-layer batch applied to `6fdfba0`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Build: `go build -trimpath -ldflags='-s -w' ./cmd/benchmark` for both engines.
- Protocol: one validation and one process-cold warmup per engine, ten alternating-order measured rounds, 250ms settle pause, 5% maximum CV.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.26.2.

Generated JSON and profiles remain local artifacts and are not committed.

## Allocation profile

Three in-process full-analysis iterations retained exact 5,357/5,357 file and 26,321-diagnostic accounting.

| Engine | Allocated space | Decision |
| --- | ---: | --- |
| Baseline `6fdfba0` | 6.61 GB | Profile reference |
| Candidate | 5.54 GB | 16.2% lower |

The prior profile's approximately 1.04 GB `copyTypeMap` site disappeared from the hot list. Persistent-layer construction and writes accounted for materially less allocation in aggregate. The next profile leaders are semantic-fact insertion, identifier folding, resolver result construction, function-scope cloning, and control-flow graph storage.

## Accepted process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.432s | 2.392s | 2.348s | 2.703s | 4.26% | 1,407,680,512 bytes | Accepted |
| Baseline `6fdfba0` | 2.511s | 2.493s | 2.455s | 2.607s | 1.87% | 1,488,617,472 bytes | Accepted |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 26,321 diagnostics.

The candidate's 2.703s maximum is retained in the full-sample gate; no outlier was removed. Both full-sample CVs pass, so the comparison supports the exact-baseline improvement.

## Correctness and boundedness

Tests cover original, clone, chained, and sibling isolation; cached class-property base sharing; a 256-entry base receiving only one delta entry; and 256 sequential clone/write cycles without exceeding the declared layer-depth bound. Missing callable-return and array-shape deletion no longer detaches shared maps.

The semantic observer retains its previous pre/post-assignment timing. An attempted shortcut changed WordPress diagnostics from 26,321 to 26,363 and was rejected before delivery.

## Decision

Retain the persistent variable/property layers and no-op deletion guards. Continue from the new profile rather than extending the representation to generic or array-index metadata without evidence. The next candidates are semantic-fact insertion, repeated normalized resolver queries, and control-flow graph storage.
