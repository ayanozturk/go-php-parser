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

Fresh reference differential evidence: PHPStan 2.2.5 level 5 passes 48/48 cases and level 7 passes 11/11 cases. The full Symfony snapshot completed (10,026 files, one parser error). The full WordPress snapshot exceeded 20 minutes and was stopped; a 100-file WordPress run and full benchmark were also stopped after several minutes. A one-file WordPress current-versus-base comparison exists, but it is not a corpus baseline. The required reproducible Symfony, WordPress, PSL, and Magento correctness/performance baseline remains open.

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
