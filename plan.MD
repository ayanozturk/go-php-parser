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
- [x] Record focused semantic correctness and microbenchmark baselines.
- [x] Capture full-registry issue snapshots for Symfony, PSL, and Magento, including parse-error accounting; collect and classify ten-run process-cold measurements for these corpora.
- [ ] Complete accepted correctness and performance baselines for the required Symfony, WordPress, PSL, and Magento corpora. The Symfony validation has one failed file; WordPress full analysis exceeded a safe memory envelope. PSL and Magento pass the current process-cold accounting and variance gates.
- [x] Separate supported behavior, known gaps, and unsupported syntax before selecting implementation slices.

Fresh focused results and current corpus evidence are recorded in [docs/benchmarks/2026-10-01-semantic-type-baseline.md](docs/benchmarks/2026-10-01-semantic-type-baseline.md). The WordPress snapshot index profile retained about 32 MB after a forced GC, while process RSS later grew past 16 GB during per-file analysis; one unrestricted attempt was killed at about 24 GB. Treat this as an analysis-phase memory problem, not an index-retention result. The next baseline action is to bound and profile per-file semantic analysis before retrying WordPress, then resolve Symfony's failed-file case.

The next refinement slice preserves `non-empty-array` through PHPDoc normalization and argument checks: populated array literals satisfy it, empty literals fail at level 5, and unrefined `array` variables fail at level 7. PHPStan 2.2.5 differential packs pass 50/50 cases at level 5 and 12/12 at level 7, including clean and mismatch controls for the new boundaries.

Integer literal, sign, and interval refinements now distinguish positive, negative, non-negative, non-positive, and bounded `int<min,max>` values in argument checks. The PHPStan 2.2.5 differential packs pass 64/64 cases at level 5, 14/14 at level 7, and 48/48 at level 3. Direct integer, non-empty-string, and non-empty-array literals are checked against refined return declarations; string and array literal emptiness and integer literals survive direct assignments; strict empty-string guards narrow nullable strings; and integer comparisons with zero or literal bounds narrow both branches, including chained `&&` comparisons and reversed operands. Exact integer and interval refinements also survive simple variable copies. Zero is rejected by `positive-int`, out-of-range literals are rejected by bounded ranges, and unrefined integer arguments fail at level 7. Broader control-flow, corpus, and type-system gates remain open.

### 2. Complete type semantics in gated slices

- [x] Infer scalar, quoted-string-literal, and nested scalar values from array-shape offsets; gate clean and mismatching call arguments at PHPStan level 5.
- [x] Preserve `non-empty-string` through PHPDoc/native merging and reject empty literals at level 5 and unrefined strings at level 7, with PHPStan 2.2.5 differential coverage.
- [x] Enforce `non-empty-array`, integer sign/range arguments, literal returns, comparisons, and simple-copy narrowing with clean, mismatch, and boundary coverage in focused tests and level 3/5/7 differential suites.
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
