# PHPStan-compatible rules: level 0

<!-- rule-inventory: level=0 introduced=1 cumulative=1 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level0.Symbols`.
- **Cumulative registered levelled rules:** 1.
- **Diagnostic families emitted:** `Level0.Symbols`, `Level0.ClassModel`, `Level0.Invocation`, and `Level0.Language`. These are emitted by the level-0 rule entry; they are not additional registered levelled rules.
- **Checked-in differential pack:** 88 cases in `testdata/diagnostic-differential`.

## Coverage and boundaries

Level 0 provides the baseline symbol, class-model, invocation, and language checks. Coverage includes selected unknown classes, interfaces, functions, methods, properties, constants, attributes, imports, type references, visibility and declaration legality, known-call argument counts, and selected invalid language constructs.

The implementation is intentionally partial compared with PHPStan level 0. Dynamic names, complete built-in and extension signatures, every modern syntax surface, all magic-member behavior, and PHPDoc references are not fully covered. Higher-level type, variable-flow, dead-code, and deprecation checks are excluded when `analysis_level: 0` is selected.
