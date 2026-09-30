# Current implementation plan

Status: in progress

Roadmap source: short-term item 2

Scope: complete the semantic and type system

The preceding compatibility plan was delivered on `main`. This plan is now active.

## Objective

Complete the core PHPDoc type model and its interactions with inference, subtyping, flow analysis, project summaries, and extension contracts while preserving executable correctness gates.

## Work sequence

### 1. Establish the current boundary

- [x] Inventory implemented type forms, normalization, substitution, narrowing, control-flow, and extension hooks against the capability matrix and current code.
- [ ] Record reproducible correctness and performance baselines for focused semantic suites and the required Symfony, WordPress, PSL, and Magento corpora.
- [x] Separate supported behavior, known gaps, and unsupported syntax before selecting implementation slices.

Initial inspection found that `analyse/generics.go` retained array-shape field types only when the field contained a class atom. Scalar, literal-string, and nested scalar offsets therefore inferred as `mixed`, despite existing shape support for callable/object offsets. The baseline `GOWORK=off go test ./...` passed; the level-5 differential pack had 41 cases before the scalar-shape fix. A PSL level-5 comparison exposed a false positive from an unbound `Tv` class-template parameter and a string-union regression; both are covered by focused tests, and `non-empty-lowercase-string` remains normalized to `string` until case-aware semantics are implemented.

Fresh reference differential evidence: PHPStan 2.2.5 level 5 passes 48/48 cases and level 7 passes 11/11 cases. The Symfony semantic snapshot completed for 10,026 files (one parser error); after the corpus runner switched to a streamed index and per-file analysis, all issue entries matched its pre-refactor output exactly. The prior scalar-shape snapshot differs from the current output in 58 Symfony files only because the new string refinement changes the displayed inferred type from `string` to `non-empty-string`; codes and locations are unchanged. The WordPress semantic snapshot did not complete: the streamed index reached an 11.6 GB physical footprint (about 9.9 GB swapped) after five minutes on this host and was stopped. This follows an earlier full-snapshot attempt over 20 minutes and a 100-file attempt over four minutes. Existing syntax identity evidence covers 5,362/5,362 WordPress files, but is not semantic or performance evidence. Fresh all-rules snapshots completed for PSL (3,319 files, 18 parse-error files, 13,906 issue entries) and Magento (25,389 files, four parse-error files, 117,758 issue entries). These snapshot selections differ from earlier parser benchmark counts of 3,361 PSL files and 41,848 Magento files, so the measurements are not directly comparable and do not establish performance baselines. The required reproducible Symfony, WordPress, PSL, and Magento correctness/performance baseline remains open.

A completed 19-file WordPress sample with heap profiles at the post-index and post-analysis boundaries retained about 5.8 MB and 4.7 MB respectively; cumulative sampled allocations were about 31 MB and 40 MB. This small sample confirms the streamed index does not retain per-file ASTs at scale, but it does not explain the earlier multi-GB process footprint or establish a full-corpus memory/performance baseline. The corpus snapshot command now supports optional profiles at both boundaries for follow-up measurements.

The next refinement slice preserves `non-empty-array` through PHPDoc normalization and argument checks: populated array literals satisfy it, empty literals fail at level 5, and unrefined `array` variables fail at level 7. PHPStan 2.2.5 differential packs pass 50/50 cases at level 5 and 12/12 at level 7, including clean and mismatch controls for the new boundaries.

Integer literal, sign, and interval refinements now distinguish positive, negative, non-negative, non-positive, and bounded `int<min,max>` values in argument checks. The PHPStan 2.2.5 differential packs pass 64/64 cases at level 5, 14/14 at level 7, and 48/48 at level 3. Direct integer, non-empty-string, and non-empty-array literals are checked against refined return declarations; string and array literal emptiness and integer literals survive direct assignments; strict empty-string guards narrow nullable strings; and integer comparisons with zero or literal bounds narrow both branches, including chained `&&` comparisons and reversed operands. Exact integer and interval refinements also survive simple variable copies. Zero is rejected by `positive-int`, out-of-range literals are rejected by bounded ranges, and unrefined integer arguments fail at level 7. Broader control-flow, corpus, and type-system gates remain open.

Strict integer equality now narrows the matching branch of `$value === N` and the false branch of `$value !== N` to the literal integer, when the current type can contain that value. Its unequal branches also exclude a literal at either endpoint of a known interval (for example, `int<5, 10>` becomes `int<6, 10>` after `$value !== 5`, or `int<5, 9>` after `$value === 10` is false). For a single integer interval, interior exclusions produce a union of the ranges on either side; already-unioned input ranges remain conservative. Loose equality is intentionally unchanged because PHP coercion can match non-integer values. The level-7 fixtures include unrefined mismatches and clean controls for matching, endpoint-exclusion, and interior-exclusion branches; the full `analyse` package passes and the level-7 differential pack passes 17/17 in both engine-only mode and against PHPStan 2.2.5.

### 2. Complete type semantics in gated slices

- [x] Infer scalar, quoted-string-literal, and nested scalar values from array-shape offsets; gate clean and mismatching call arguments at PHPStan level 5.
- [x] Preserve `non-empty-string` through PHPDoc/native merging and reject empty literals at level 5 and unrefined strings at level 7, with PHPStan 2.2.5 differential coverage.
- [ ] Cover templates, bounds, variance, generic inheritance, aliases/imports, conditional types, indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, and integer ranges.
- Complete normalization, subtyping, substitution, narrowing/widening, recursion limits, and `self`/`static`/`parent`/`$this` behavior.
- Give each correction neutral positive, negative, and boundary fixtures at the relevant PHPStan level.

### 3. Extend flow and project semantics

- Extend CFG and variable/property flow through loops, exceptions, generators, closures, references, property initialization, readonly constraints, and dynamic properties.
- Add interprocedural summaries and invalidate dependants only when exported semantic facts change.
- Define executable extension contracts for dynamic returns, assertions, generated members, reflection, service containers, and ORMs.

### 4. Verify ecosystem behavior

- Run documented correctness, race, corpus, and performance gates on Symfony, WordPress, PSL, and Magento.
- Keep capability and rule documentation synchronized with measured behavior.

## Acceptance gates

- Generic, shape, callable, flow, property, trait/interface, and summary suites meet their documented gates.
- Symfony, Doctrine, Composer, PHPUnit, and Laravel compatibility packs pass.
- WordPress, Symfony, PSL, and Magento complete without panic, timeout, race, or silent omission.
- Performance and corpus claims include complete workload and file accounting.
