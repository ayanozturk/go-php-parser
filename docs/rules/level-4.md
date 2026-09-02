# PHPStan-compatible rules: level 4

<!-- rule-inventory: level=4 introduced=1 cumulative=22 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Generic.CodeAnalysis.UnreachableCode`.
- **Cumulative registered levelled rules:** 22.
- **Checked-in differential pack:** None currently checked in.

## Coverage and boundaries

`Generic.CodeAnalysis.UnreachableCode` reports selected statements that cannot execute after terminating control flow, such as `return`, `throw`, and equivalent branches recognized by the control-flow analysis. It is a focused dead-code check, not a complete reachability proof for every PHP construct or path-sensitive condition.

Levels 0 through 3 are cumulative. No PHPStan differential pack currently gates this level.
