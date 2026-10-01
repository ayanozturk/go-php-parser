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
- [ ] Cover indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, and integer ranges.
- Complete normalization, subtyping, substitution, narrowing/widening, recursion limits, and `self`/`static`/`parent`/`$this` behavior.
- Give each correction neutral positive, negative, and boundary fixtures at the relevant PHPStan level.

Immediate next slice: resolve PHPDoc indexed-access types against array shapes and generic key/value types, with clean, mismatch, and boundary fixtures.

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
