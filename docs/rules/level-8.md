# PHPStan-compatible rules: level 8

<!-- rule-inventory: level=8 introduced=1 cumulative=15 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level8.MethodNonObject`.
- **Cumulative registered levelled rules:** 15.
- **Checked-in differential pack:** 5 cases in `testdata/diagnostic-differential-level8`.

## Coverage and boundaries

`Level8.MethodNonObject` reports selected method calls where a nullable or otherwise non-object receiver can reach the call. It complements the lower-level non-object checks with the level-8 nullable-object boundary and uses inferred receiver types where available.

The rule does not model every dynamic call, magic method, or unresolved union. Levels 0 through 7 are cumulative.
