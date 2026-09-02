# PHPStan-compatible rules: level 2

<!-- rule-inventory: level=2 introduced=6 cumulative=8 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 6 registered levelled rules: `A.ASSIGN.OP.INVALID`, `A.BINARY.OP.INVALID`, `A.VOID.PURE`, `Level2.MethodExistence`, `Level2.MethodNonObject`, and `Level2.MethodVisibility`.
- **Cumulative registered levelled rules:** 8.
- **Checked-in differential pack:** 76 cases in `testdata/diagnostic-differential-level2`.

## Coverage and boundaries

Level 2 adds invalid assignment and binary-operator checks, pure named `void` function detection, and method-call analysis for typed receivers. Binary-operation coverage currently proves invalid numeric/string and array/scalar additions, with valid integer and array-union controls. `A.VOID.PURE` currently covers functions whose bodies contain only explicit returns; effectful and empty functions stay clean. Method checks report selected calls to undefined methods, methods on non-object values, and methods whose visibility is not accessible.

Receiver inference remains deliberately conservative for dynamic calls, arbitrary unresolved expressions, and complex type combinations. Numeric strings, bitwise operations, object conversions, and broader operator combinations remain outside the exact binary-operation gate. Levels 0 and 1 are cumulative.
