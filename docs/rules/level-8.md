# PHPStan-compatible rules: level 8

<!-- rule-inventory: level=8 introduced=1 cumulative=35 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level8.MethodNonObject`.
- **Cumulative registered levelled rules:** 35.
- **Checked-in differential pack:** 51 cases in `testdata/diagnostic-differential-level8`.

## Registered rules, one by one

### `Level8.MethodNonObject`

**What it checks:** Reports selected method calls where nullable or otherwise non-object receiver values can reach the call.

**Why it helps:** A nullable receiver can cause a runtime error unless the code proves it is an object first.

**Example that reports:**

```php
<?php
function save(?Repository $repo): void { $repo->flush(); }
```

**A safer form:**

```php
<?php
function save(?Repository $repo): void { if ($repo !== null) { $repo->flush(); } }
```

## Coverage and boundaries

`Level8.MethodNonObject` reports selected method calls where a nullable or otherwise non-object receiver can reach the call. It complements the lower-level non-object checks with the level-8 nullable-object boundary and uses inferred receiver types where available. Nullable-only argument mismatches also begin at level 8 in the cumulative `A.ARG.TYPE` checks; the clean level-7 boundary and failing level-8 control pin that threshold.

The rule does not model every dynamic call, magic method, or unresolved union. The `known-method-negated-instanceof-or` clean control proves a known method on the right of `!$x instanceof T || …` after false-scope narrowing. True-scope `method_exists`/`is_object` on property-fetch receivers (`$this->prop`, `$obj->prop`) keeps guarded calls clean in `if`, ternary, and `&&` forms, using property types broad enough that the guards are not redundant under PHPStan. Other controls cover a boolean variable built from an independent flag and a nullable-property check, a PHPUnit equality assertion followed by an explicit non-null assertion, and guarded `getPrevious()` access. The 51-case pack also pins nullable argument mismatches at their PHPStan level-8 threshold, including Symfony service-container and Laravel Eloquent builder results, and exercises Composer's nullable installed-version result. Levels 0 through 7 are cumulative.
