# Current implementation plan

Status: ready

Roadmap source: short-term item 2 — complete the semantic and type system

## Next slice

Carry exact quoted PHPDoc string literal types through expression-level value selection (ternary, match, and null-coalescing expressions) so later function, method, and constructor arguments retain the selected values. Keep the work scoped to expression inference and existing local flow; do not expand callable signature or class-constant `value-of` handling in this slice.

Preserve semantic inferred-type facts as authoritative and retain the level-specific broad-string behavior verified against PHPStan 2.2.5.

## Acceptance evidence

- Add neutral PHPStan 2.2.5 differential fixtures for ternary, match, and null-coalescing expressions with compatible and incompatible literal argument calls.
- Confirm the level threshold for every expression form and add below-level controls where applicable.
- Preserve exact assignment/join/narrowing, return/property propagation, finite-shape `value-of`, and explicit semantic-fact behavior.
- Add focused unit tests for selected literal expression types and unions.
- Run the full Go suite and affected PHPStan reference comparisons.
- Update the capability matrix and affected rule documentation.
- Commit and push the completed slice.

## Parent objective

Continue roadmap item 2: complete PHPDoc templates, bounds, variance, generic inheritance, aliases/imports, conditional types, indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, integer ranges, and their interactions with flow and project semantics.
