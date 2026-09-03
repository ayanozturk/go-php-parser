# PHPStan-compatible rules: level 5

<!-- rule-inventory: level=5 introduced=1 cumulative=24 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `A.ARG.TYPE`.
- **Cumulative registered levelled rules:** 24.
- **Checked-in differential pack:** 6 cases in `testdata/diagnostic-differential-level5`.

## Coverage and boundaries

`A.ARG.TYPE` reports incompatible arguments passed to resolved named functions, methods, and constructors. The six-case differential pack covers positional and named mismatches plus compatible scalar and subtype controls; all four failing cases are silent at level 4 and match PHPStan's `argument.type` identifier at level 5. Static methods, dynamic callables, unpacked arguments, extension-provided signatures, and the full PHPStan type lattice remain partial. The rule uses the existing shared argument-call traversal, so moving it from level 10 does not add a pass to the default analysis path.
