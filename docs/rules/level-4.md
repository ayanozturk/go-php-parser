# PHPStan-compatible rules: level 4

<!-- rule-inventory: level=4 introduced=1 cumulative=27 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Generic.CodeAnalysis.UnreachableCode`.
- **Cumulative registered levelled rules:** 27.
- **Checked-in differential pack:** 7 cases in `testdata/diagnostic-differential-level4`.

## Registered rules, one by one

### `Generic.CodeAnalysis.UnreachableCode`

**What it checks:** Reports selected statements after a terminating return or throw that cannot be reached.

**Why it helps:** Unreachable statements never run and can hide misplaced logic or stale code.

**Example that reports:**

```php
<?php
function value(): int {
    return 1;
    echo "never";
}
```

**A safer form:**

```php
<?php
function value(): int {
    echo "before return";
    return 1;
}
```

## Coverage and boundaries

`Generic.CodeAnalysis.UnreachableCode` reports selected statements that cannot execute after terminating control flow, such as `return` and `throw`. The pinned PHPStan 2.2.5 differential pack covers both terminators, a clean conditional-return control, and confirms that `value-of` and quoted PHPDoc string argument mismatches, including assignment, branch-join, and strict-comparison controls, remain silent at level 4 before argument checking begins at level 5. Exact PHPDoc string return and property mismatches report through the level-3 return/property checks; their argument mismatch control remains gated until level 5. This is a focused dead-code check, not a complete reachability proof for every PHP construct or path-sensitive condition.

Levels 0 through 3 are cumulative.
