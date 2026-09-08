# PHPStan-compatible rules: level 9

<!-- rule-inventory: level=9 introduced=1 cumulative=33 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `A.DEPRECATED.CALL`.
- **Cumulative registered levelled rules:** 33.
- **Checked-in differential pack:** None currently checked in.

## Coverage and boundaries

Level 9 adds warning diagnostics for selected deprecated calls. Deprecation checking uses available declaration metadata and reports warnings rather than errors. Levels 0 through 8 are cumulative.

PHPStan level 9 mixed-type coverage is not implemented. The rule is partial compared with PHPStan's complete signature and deprecation metadata. Dynamic calls and extension-dependent metadata remain outside the covered subset.
