# PHPStan-compatible rules: level 7

<!-- rule-inventory: level=7 introduced=1 cumulative=24 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level7.MethodUnion`.
- **Cumulative registered levelled rules:** 24.
- **Checked-in differential pack:** 5 cases in `testdata/diagnostic-differential-level7`.

## Coverage and boundaries

`Level7.MethodUnion` reports selected method calls that are not available on every member of a union or disjunctive-normal-form receiver type. It relies on resolved receiver metadata and is intentionally conservative when a union member cannot be resolved or when magic and dynamic methods affect the result.

Levels 0 through 4 are cumulative. Level 5 and level 6 currently add no registered rules.
