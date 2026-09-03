# Unlevelled analysis rules

<!-- rule-inventory: unlevelled=4 levelled=33 total=37 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

These rules are registered without a PHPStan-compatible analysis level:

- `A.ARG.COUNT`
- `Generic.CodeAnalysis.AssignmentInCondition`
- `Generic.CodeAnalysis.EmptyStatement`
- `PSR1.Files.SideEffects`

- **Introduced outside an exact level:** 4 registered unlevelled rules.
- **Cumulative registered levelled rules:** 33 (unchanged; these rules are not part of that total).
- **Total registered analysis rules:** 37, consisting of 33 levelled rules and these 4 unlevelled rules.
- **Checked-in differential pack:** None currently checked in.

The four unlevelled rules run only when `analysis_level` is unset. When an explicit analysis level is selected, only registered rules at or below that level are enabled.

## Coverage and boundaries

`A.ARG.COUNT` is the legacy argument-count rule for resolved method and constructor calls. The level-aware level-0 invocation checks are used instead when explicit level mode is selected.

`Generic.CodeAnalysis.AssignmentInCondition` reports assignments used as conditions, `Generic.CodeAnalysis.EmptyStatement` reports selected empty statements, and `PSR1.Files.SideEffects` reports file-level side effects under the existing style-rule behavior. These rules are retained for compatibility and are not counted in any PHPStan level's cumulative levelled total.
