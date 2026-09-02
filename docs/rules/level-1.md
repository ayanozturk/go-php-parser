# PHPStan-compatible rules: level 1

<!-- rule-inventory: level=1 introduced=1 cumulative=2 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level1.Variables`.
- **Cumulative registered levelled rules:** 2.
- **Checked-in differential pack:** 24 cases in `testdata/diagnostic-differential-level1`.

## Coverage and boundaries

`Level1.Variables` reports always-undefined and possibly-undefined variable reads using joined flow facts. It covers common branches, short-circuit expressions, ternaries, bounded loops, `switch`, `try`/`catch`/`finally`, globals and statics, destructuring, closure captures, references, selected by-reference outputs, known-string dynamic reads, `extract`, `compact`, and `isset`/`empty` suppression.

Dynamic calls, dynamic transfer levels, complex dynamic-name expressions, and extension-dependent built-in signatures remain conservative or incomplete. Level 1 includes the cumulative level-0 rule set.
