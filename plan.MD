# Current implementation plan

Status: in progress

Roadmap source: short-term item 2 — complete the semantic and type system

## Current slice

Carry exact quoted PHPDoc string literals through local variable assignments and common flow joins/narrowing so later function, method, and constructor arguments retain their exact values. Preserve semantic inferred-type facts as authoritative and retain the level-specific broad-string behavior verified against PHPStan 2.2.5.

Keep this slice focused on local flow. Callable signatures, class-constant `value-of`, and more complex conditional types remain future work.

## Acceptance evidence

- Add neutral PHPStan 2.2.5 differential fixtures for exact literal assignment, compatible and incompatible argument calls, and branch/assignment-join behavior.
- Confirm which flow cases preserve exact literals at level 5 and add below-level controls where applicable.
- Preserve current exact-literal return/property propagation, finite-shape `value-of`, and explicit semantic-fact behavior.
- Add focused unit tests for assignment, joins, and narrowing.
- Run the full Go suite and affected PHPStan reference comparisons.
- Update the capability matrix and affected rule documentation.
- Commit and push the completed slice.

## Parent objective

Continue roadmap item 2: complete PHPDoc templates, bounds, variance, generic inheritance, aliases/imports, conditional types, indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, integer ranges, and their interactions with flow and project semantics.
