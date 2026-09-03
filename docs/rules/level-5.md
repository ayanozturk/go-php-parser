# PHPStan-compatible rules: level 5

<!-- rule-inventory: level=5 introduced=1 cumulative=25 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `A.ARG.TYPE`.
- **Cumulative registered levelled rules:** 25.
- **Checked-in differential pack:** 7 cases in `testdata/diagnostic-differential-level5`.

## Coverage and boundaries

`A.ARG.TYPE` reports incompatible arguments passed to resolved named functions, methods, and constructors. The seven-case differential pack covers positional and named mismatches plus compatible scalar, subtype, and PHP 8.3+ `DateTime::modify()` return controls; all four failing cases are silent at level 4 and match PHPStan's `argument.type` identifier at level 5. Function/method-local PHPDoc template parameters are treated as conservative `mixed` at call sites until call-site template binding is available, preventing unresolved template names from becoming false concrete-class mismatches. Static methods, dynamic callables, unpacked arguments, extension-provided signatures, alias-body expansion, call-site template binding, and the full PHPStan type lattice remain partial. The rule uses the existing shared argument-call traversal, so moving it from level 10 does not add a pass to the default analysis path.
