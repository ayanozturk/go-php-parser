# Current implementation plan

Status: in progress

Roadmap source: short-term item 2

Scope: complete the semantic and type system

The preceding compatibility plan was delivered on `main`. This plan is now active.

## Objective

Complete the core PHPDoc type model and its interactions with inference, subtyping, flow analysis, project summaries, and extension contracts while preserving executable correctness gates.

## Work sequence

### 2. Complete type semantics in gated slices

- [x] Preserve declaration-site template variance and check nested generic/callable parameter and return positions; enforce generic bound compatibility through declared variance and inherited `@extends`/`@implements` substitutions.
- [x] Expand nested local `@phpstan-type`/`@psalm-type` aliases through generic and callable types, resolve imported class aliases in alias bodies, and retain exact token boundaries with clean, mismatch, and boundary fixtures.
- [x] Model parameter-dependent conditional PHPDoc return types, select branches from call arguments, and feed the result into argument checking with clean, mismatch, and unresolved-condition boundary fixtures.
- [x] Resolve PHPDoc indexed access against array-shape literal keys and generic array key/value types, retaining unresolved `T[K]` expressions; add clean, mismatch, and boundary fixtures.
- [x] Resolve `key-of`/`value-of` projections for array shapes and generic key/value arrays while preserving unresolved template projections; add clean, mismatch, and boundary fixtures.
- [x] Flow callable parameter and return signatures through `callable`/`Closure` invocations, checking incompatible arguments while keeping mixed-value boundaries clean.
- [x] Preserve optional and nested array-shape fields during indexed access and project value unions across union-shaped arrays, with clean, mismatch, and boundary fixtures.
- [x] Preserve quoted PHPDoc string literals as exact types through shape values, `key-of` projections, assignments, hover, and return diagnostics, with clean, mismatch, and boundary fixtures.
- [x] Complete non-empty string refinements and integer-range boundaries with clean, mismatch, and boundary fixtures.
- Complete normalization, subtyping, substitution, narrowing/widening, recursion limits, and `self`/`static`/`parent`/`$this` behavior.
- [x] Keep generic template substitution out of quoted PHPDoc string literals while binding surrounding shape types.
- [x] Resolve PHPDoc `self`, `static`, `parent`, and `$this` method return types at call sites, preserving the declaring and receiver class boundaries.
- Give each correction neutral positive, negative, and boundary fixtures at the relevant PHPStan level.

Immediate next slice: verify generic normalization and subtyping across nested unions/intersections, including inherited substitutions and unresolved-template boundaries.

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
