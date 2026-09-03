# PHPStan-compatible rules: level 7

<!-- rule-inventory: level=7 introduced=1 cumulative=31 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level7.MethodUnion`.
- **Cumulative registered levelled rules:** 31.
- **Checked-in differential pack:** 6 cases in `testdata/diagnostic-differential-level7`.

## Coverage and boundaries

`Level7.MethodUnion` reports selected method calls that are not available on every member of a union or disjunctive-normal-form receiver type. It relies on resolved receiver metadata and is intentionally conservative when a union member cannot be resolved or when magic and dynamic methods affect the result. The differential pack also verifies cumulative strict argument checking for a `DateTime|false` return union while keeping PHP 8.3+ `DateTime::modify()` non-false.

Levels 0 through 6 are cumulative. Level 5 adds argument-type checks; level 6 adds focused missing-type checks.
