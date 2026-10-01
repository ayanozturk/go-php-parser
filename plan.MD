# Current implementation plan

Status: in progress

Roadmap source: short-term item 2 — complete the semantic and type system

## Current slice

Carry exact quoted PHPDoc string literals through function/method return types, typed properties, and semantic inferred-type facts so callers can use literal precision at argument boundaries. Preserve explicit semantic facts as authoritative and keep general `string` behavior consistent with PHPStan 2.2.5.

Keep this slice focused on return/property propagation. Callable signatures, class-constant `value-of`, and broader flow/conditional type behavior remain future work.

## Acceptance evidence

- Add neutral PHPStan 2.2.5 differential fixtures for literal return and property values, including clean and mismatched argument controls.
- Confirm the reporting level for return/property mismatches with the pinned reference analyser.
- Preserve existing exact-literal unions, finite-shape `value-of`, and semantic inferred-type fact behavior.
- Add focused unit tests for literal propagation through returns and properties.
- Run the full Go suite and the affected PHPStan reference comparisons.
- Update the capability matrix and affected rule documentation.
- Commit and push the completed slice.

## Parent objective

Continue roadmap item 2: complete PHPDoc templates, bounds, variance, generic inheritance, aliases/imports, conditional types, indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, integer ranges, and their interactions with flow and project semantics.
