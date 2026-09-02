# PHPStan-compatible rules: level 6

<!-- rule-inventory: level=6 introduced=5 cumulative=28 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 5 registered levelled rules: `Level6.MissingGenericType`, `Level6.MissingIterableValueType`, `Level6.MissingParameterType`, `Level6.MissingReturnType`, and `Level6.MissingPropertyType`.
- **Cumulative registered levelled rules:** 28.
- **Checked-in differential pack:** 16 cases in `testdata/diagnostic-differential-level6`.

## Coverage and boundaries

Level 6 adds focused missing-type checks for generic class arguments, iterable value types, and untyped declarations. `Level6.MissingGenericType` reports generic classes used without template arguments in selected parameter and return declarations. `Level6.MissingIterableValueType` reports `array` and `iterable` declarations without value types, including selected PHPDoc array forms. `Level6.MissingParameterType`, `Level6.MissingReturnType`, and `Level6.MissingPropertyType` report missing parameter, return, and property types while respecting PHPDoc declarations, explicit `mixed`, and constructor exemptions. The 16-case differential pack covers the original generic/iterable failures and controls plus named-function and method declaration failures, a property failure, and clean mixed/void and PHPDoc controls; all ten failing cases remain silent at level 5. Broader missing-type inference for aliases, conditional types, dynamic or extension-provided types, and every iterable/generic declaration remains outside this exact gate. Selecting level 6 runs the cumulative rule set from levels 0 through 5.
