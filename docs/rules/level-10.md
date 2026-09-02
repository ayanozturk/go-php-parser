# PHPStan-compatible rules: level 10

<!-- rule-inventory: level=10 introduced=2 cumulative=27 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 2 registered levelled rules: `A.ARG.TYPE` and `A.DEPRECATED.CALL`.
- **Cumulative registered levelled rules:** 27.
- **Checked-in differential pack:** None currently checked in.

## Coverage and boundaries

Level 10 adds argument-type compatibility checks and warning diagnostics for selected deprecated calls. Argument checking covers inferred expressions, selected union and null refinements, inherited signatures, named arguments, and known built-ins. Deprecation checking uses available declaration metadata and reports warnings rather than errors.

Both rules are partial compared with PHPStan's complete type system, signature database, and deprecation metadata. Dynamic calls, extension-dependent signatures, PHPDoc-only declarations, and unresolved expressions remain outside the covered subset. All lower levels are cumulative.
