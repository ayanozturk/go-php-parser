# PHPStan-compatible rules: level 7

<!-- rule-inventory: level=7 introduced=1 cumulative=34 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level7.MethodUnion`.
- **Cumulative registered levelled rules:** 32.
- **Checked-in differential pack:** 11 cases in `testdata/diagnostic-differential-level7`.

## Coverage and boundaries

`Level7.MethodUnion` reports selected method calls that are not available on every member of a union or disjunctive-normal-form receiver type. It relies on resolved receiver metadata and is intentionally conservative when a union member cannot be resolved or when magic and dynamic methods affect the result. PHPUnit `T|MockObject` is treated as `T&MockObject` for method existence, and class-scope method return types use DNF strings so intersections are not flattened to unions. Clean controls include `known-method-phpunit-mock-intersection` and `known-method-phpunit-create-mock`. The differential pack also verifies argument checking for an extra `false` union arm and a general `string` passed to `non-empty-string`: PHPStan first reports this mismatch at level 7, while nullable-only argument mismatches remain clean until level 8. It keeps PHP 8.3+ `DateTime::modify()` non-false.

Levels 0 through 6 are cumulative. Level 5 adds argument-type checks; level 6 adds focused missing-type checks.
