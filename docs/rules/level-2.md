# PHPStan-compatible rules: level 2

<!-- rule-inventory: level=2 introduced=5 cumulative=7 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 5 registered levelled rules: `A.ASSIGN.OP.INVALID`, `A.VOID.PURE`, `Level2.MethodExistence`, `Level2.MethodNonObject`, and `Level2.MethodVisibility`.
- **Cumulative registered levelled rules:** 7.
- **Checked-in differential pack:** 72 cases in `testdata/diagnostic-differential-level2`.

## Coverage and boundaries

Level 2 adds invalid assignment-operator checks, pure named `void` function detection, and method-call analysis for typed receivers. `A.VOID.PURE` currently covers functions whose bodies contain only explicit returns; effectful and empty functions stay clean. Method checks report selected calls to undefined methods, methods on non-object values, and methods whose visibility is not accessible.

Receiver inference remains deliberately conservative for dynamic calls, arbitrary unresolved expressions, and complex type combinations. Compound assignment support and broader expression-result typing continue to expand incrementally. Levels 0 and 1 are cumulative.
