# PHPStan-compatible rules: level 10

<!-- rule-inventory: level=10 introduced=1 cumulative=32 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `A.DEPRECATED.CALL`.
- **Cumulative registered levelled rules:** 32.
- **Checked-in differential pack:** None currently checked in.

## Coverage and boundaries

Level 10 adds warning diagnostics for selected deprecated calls. Deprecation checking uses available declaration metadata and reports warnings rather than errors. Argument-type compatibility is introduced cumulatively at level 5.

The rule is partial compared with PHPStan's complete signature and deprecation metadata. Dynamic calls and extension-dependent metadata remain outside the covered subset. All lower levels are cumulative.
