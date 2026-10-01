# PHPStan-compatible rules: level 4

<!-- rule-inventory: level=4 introduced=1 cumulative=25 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Generic.CodeAnalysis.UnreachableCode`.
- **Cumulative registered levelled rules:** 25.
- **Checked-in differential pack:** 4 cases in `testdata/diagnostic-differential-level4`.

## Coverage and boundaries

`Generic.CodeAnalysis.UnreachableCode` reports selected statements that cannot execute after terminating control flow, such as `return` and `throw`. The pinned PHPStan 2.2.5 differential pack covers both terminators, a clean conditional-return control, and confirms that `value-of` argument mismatches remain silent at level 4 before argument checking begins at level 5. This is a focused dead-code check, not a complete reachability proof for every PHP construct or path-sensitive condition.

Levels 0 through 3 are cumulative.
