# PHPStan-compatible rules: level 3

<!-- rule-inventory: level=3 introduced=5 cumulative=12 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 5 registered levelled rules: `A.PROP.TYPE`, `A.RETURN.NEVER`, `A.RETURN.TYPE`, `A.RETURN.VOID`, and `Level3.ThrowType`.
- **Cumulative registered levelled rules:** 12. `A.ASSIGN.OP.INVALID` and `A.VOID.PURE` are cumulative from level 2 and are not introduced here.
- **Checked-in differential pack:** 22 cases in `testdata/diagnostic-differential-level3`.

## Coverage and boundaries

Level 3 checks declared return types, values returned from `void` functions and methods, `never` return contracts, typed-property assignments, and selected throw targets. Return and property checks use inferred expression types, immutable semantic facts, and known class/property metadata where available.

Coverage remains partial for PHPDoc types, dynamic expressions, complex unions, every assignment operator, and the complete PHPStan type system. Levels 0 through 2 are cumulative.
