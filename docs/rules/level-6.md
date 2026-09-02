# PHPStan-compatible rules: level 6

<!-- rule-inventory: level=6 introduced=2 cumulative=25 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 2 registered levelled rules: `Level6.MissingGenericType` and `Level6.MissingIterableValueType`.
- **Cumulative registered levelled rules:** 25.
- **Checked-in differential pack:** 8 cases in `testdata/diagnostic-differential-level6`.

## Coverage and boundaries

Level 6 adds focused missing-type checks for generic class arguments and iterable value types. `Level6.MissingGenericType` reports generic classes used without template arguments in selected parameter and return declarations. `Level6.MissingIterableValueType` reports `array` and `iterable` declarations without value types, including selected PHPDoc array forms. The eight-case differential pack covers failing native/PHPDoc declarations plus generic, typed-array, and array-shape controls; the failing cases remain silent at level 5. Broader missing-type inference for aliases, conditional types, dynamic or extension-provided types, and every iterable/generic declaration remains outside this exact gate. Selecting level 6 runs the cumulative rule set from levels 0 through 5.
