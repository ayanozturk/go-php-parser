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
- [x] Capture full-registry issue snapshots and classify ten-run process-cold measurements for Symfony, WordPress, PSL, and Magento.
- [x] Complete accepted correctness and performance baselines for the required Symfony, WordPress, PSL, and Magento corpora, with exact accounting for the known Symfony parse-error fixture.
- [x] Separate supported behavior, known gaps, and unsupported syntax before selecting implementation slices.

The completed baseline and focused semantic evidence are recorded in [docs/benchmarks/2026-10-01-semantic-type-baseline.md](docs/benchmarks/2026-10-01-semantic-type-baseline.md). The baseline exposed exponential path growth in optional guaranteed-loop-exit refinement; a shared 128-path budget now bounds both AST and CST simulation. The four required corpora complete with accounted workloads, and the changed analyzer has zero issue-set mismatches on the previously captured PSL, Symfony, and Magento snapshots.

### 2. Complete type semantics in gated slices

- [x] Infer scalar, quoted-string-literal, and nested scalar values from array-shape offsets; gate clean and mismatching call arguments at PHPStan level 5.
- [x] Preserve `non-empty-string` through PHPDoc/native merging and reject empty literals at level 5 and unrefined strings at level 7, with PHPStan 2.2.5 differential coverage.
- [x] Enforce `non-empty-array`, integer sign/range arguments, literal returns, comparisons, and simple-copy narrowing with clean, mismatch, and boundary coverage in focused tests and level 3/5/7 differential suites.
- [ ] Cover templates, bounds, variance, generic inheritance, aliases/imports, conditional types, indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, and integer ranges.
- Complete normalization, subtyping, substitution, narrowing/widening, recursion limits, and `self`/`static`/`parent`/`$this` behavior.
- Give each correction neutral positive, negative, and boundary fixtures at the relevant PHPStan level.

Immediate next slice: close template bounds, variance, and generic inheritance/substitution with clean, mismatch, and boundary fixtures before expanding the remaining PHPDoc forms.

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
