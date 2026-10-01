# Current implementation plan

Status: in progress

Roadmap source: short-term item 2 — complete the semantic and type system

## Current slice

Preserve quoted PHPDoc string literals as exact types through normalization, union handling, and function/method/constructor argument compatibility. Keep literal strings distinct from general `string` while ensuring every literal remains a subtype of `string`.

Keep this slice focused on string literals at argument boundaries. Literal-string return/property inference and projections beyond the finite-shape `value-of` support are future work.

## Acceptance evidence

- Add neutral PHPStan 2.2.5 differential fixtures for exact literal acceptance, wrong-literal rejection, literal unions, and general-string clean/mismatch controls.
- Confirm the mismatch threshold with a clean level-4 fixture and a level-5 reporting fixture.
- Preserve existing PHPDoc array-shape offsets and `value-of<array{...}>` argument cases.
- Add focused unit tests for literal-string normalization and subtype behavior.
- Run the full Go suite, level-4/5 engine differential packs, and their PHPStan reference comparisons.
- Update the capability matrix and affected level documentation.
- Commit and push the completed slice.

## Parent objective

Continue roadmap item 2: complete PHPDoc templates, bounds, variance, generic inheritance, aliases/imports, conditional types, indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, integer ranges, and their interactions with flow and project semantics.
