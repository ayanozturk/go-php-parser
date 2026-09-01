# WordPress shared function-scope context, 2026-09-01

## Outcome

Branch-sensitive function scopes now keep immutable class metadata and `FileTypeContext` map headers behind one shared context pointer. Clones still own their shallow mutable state and retain the existing copy-on-write layers/maps for variables, properties, callable returns, array shapes, array-index keys, and generic instances.

This batch is accepted as an allocation improvement against exact baseline `8f400be`. The process-cold comparison is rejected because both engines exceeded the 5% CV contract. This is not a Mago comparison.

## Reproducible source and workload

- Baseline engine commit: `8f400be53b88be0f53ddbcd92ed1411a2ef6521f`.
- Candidate: this shared function-scope context batch applied to `8f400be`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every registered rule, eight workers.
- Allocation protocol: three in-process full-analysis iterations with `--memprofile`.
- Cold protocol: one validation and one process-cold warmup per engine, ten alternating-order measured rounds plus ten extra measured runs, 250ms settle pause, 5% maximum CV.
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.27.0.

Generated JSON and profiles remain local artifacts and are not committed.

## Focused clone benchmark

Five samples of each benchmark retained one allocation for a read-only clone and three for clone-plus-write.

| Operation | Baseline | Candidate | Allocated-space change |
| --- | ---: | ---: | ---: |
| Read-only clone | 208 B/op, 43.74–44.52 ns/op | 96 B/op, 26.68–26.97 ns/op | 53.8% lower |
| Clone plus variable/property write | 368 B/op, 91.04–92.95 ns/op | 256 B/op, 65.85–66.59 ns/op | 30.4% lower |

Adversarial tests cap the scope value at 96 bytes on 64-bit builds, require at most one read-only clone allocation, prove the immutable context and nested metadata maps stay shared, and verify every mutable state family detaches on write.

## Allocation profile

Both profiles used three in-process full-analysis iterations. Every iteration parsed 5,357/5,357 files and emitted exactly 22,387 diagnostics.

| Engine | Allocated space | `functionScope.clone` | Decision |
| --- | ---: | ---: | --- |
| Baseline `8f400be` | 3,516,256,187 bytes | 401,159,880 bytes | Exact committed baseline |
| Candidate | 3,273,965,861 bytes | 173,555,216 bytes | 6.9% / 56.7% lower |

The next allocation leaders are control-flow scope/reachability maps and generated semantic-fact insertion. The scope clone is no longer the largest mutable-analysis allocation site.

## Rejected process-cold comparison

| Engine | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 2.398s | 2.146s | 2.023s | 4.020s | 23.39% | 1,136,164,864 bytes | Rejected |
| Baseline `8f400be` | 2.269s | 2.139s | 2.037s | 3.984s | 18.04% | 1,149,616,128 bytes | Rejected |

Every validation, warmup, and measured run discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero failures and exactly 22,387 diagnostics. Neither the raw timing nor RSS direction is accepted performance evidence because neither engine passed the stability gate.

## Decision

Retain the shared immutable context because it removes a measured per-clone allocation cost without changing ownership or mutation semantics. Continue with the remaining control-flow scope and reachability maps, then rerun an isolated-host exact-baseline and contemporaneous Mago comparison.
