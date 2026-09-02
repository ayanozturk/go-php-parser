# PHPStan-compatible rules: level 2

<!-- rule-inventory: level=2 introduced=14 cumulative=16 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 14 registered levelled rules: `A.ASSIGN.OP.INVALID`, `A.BINARY.OP.INVALID`, `A.VOID.PURE`, `Level2.MethodExistence`, `Level2.MethodNonObject`, `Level2.MethodVisibility`, `Level2.PHPDocClass`, `Level2.PHPDocGenericLessTypes`, `Level2.PHPDocGenericMoreTypes`, `Level2.PHPDocNotGeneric`, `Level2.PHPDocParamName`, `Level2.PHPDocParamType`, `Level2.PHPDocPropertyType`, and `Level2.PHPDocReturnType`.
- **Cumulative registered levelled rules:** 16.
- **Checked-in differential pack:** 90 cases in `testdata/diagnostic-differential-level2`.

## Coverage and boundaries

Level 2 adds invalid assignment and binary-operator checks, PHPDoc validation, pure named `void` function detection, and method-call analysis for typed receivers. PHPDoc coverage checks simple and nested generic class references, excess or missing generic arguments on indexed template classes, generic arguments applied to non-generic classes, tags for missing parameters, and parameter, return, or property annotations incompatible with native declarations. All eight PHPDoc rules reuse the existing structural traversal. Binary-operation coverage currently proves invalid numeric/string and array/scalar additions, with valid integer and array-union controls. `A.VOID.PURE` currently covers functions whose bodies contain only explicit returns; effectful and empty functions stay clean. Method checks report selected calls to undefined methods, methods on non-object values, and methods whose visibility is not accessible.

Receiver inference remains deliberately conservative for dynamic calls, arbitrary unresolved expressions, and complex type combinations. PHPDoc shapes, callable-signature internals, variance, bounds, aliases beyond the existing type normalizer, and full tag legality remain partial; template-bearing native-compatibility checks stay conservative. Numeric strings, bitwise operations, object conversions, and broader operator combinations remain outside the exact binary-operation gate. Levels 0 and 1 are cumulative.
