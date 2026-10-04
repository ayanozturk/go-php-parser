# PHPStan-compatible rules: level 0

<!-- rule-inventory: level=0 introduced=2 cumulative=2 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 2 registered levelled rules: `Level0.PropertyCallableType` and `Level0.Symbols`.
- **Cumulative registered levelled rules:** 2.
- **Diagnostic families emitted:** `Level0.Symbols`, `Level0.ClassModel`, `Level0.Invocation`, and `Level0.Language`. These are emitted by the level-0 rule entry; they are not additional registered levelled rules.
- **Checked-in differential pack:** 98 cases in `testdata/diagnostic-differential` (96 PHPStan differential cases and 2 explicit unsupported boundaries).

## Registered rules, one by one

### `Level0.PropertyCallableType`

**What it checks:** Flags native callable property declarations, including nullable, union, and promoted properties.

**Why it helps:** Native callable properties have runtime and initialization constraints that are easy to overlook; use a Closure property or an explicit method API when that is the intent.

**Example that reports:**

```php
<?php
class Handler {
    public callable $callback;
}
```

**A safer form:**

```php
<?php
class Handler {
    public Closure $callback;
}
```

### `Level0.Symbols`

**What it checks:** Checks selected class-like, function, method, property, constant, attribute, import, and type references against the project symbol index.

**Why it helps:** Catching misspelled or missing declarations before runtime prevents avoidable fatal errors and broken references.

**Example that reports:**

```php
<?php
new MissingService();
```

**A safer form:**

```php
<?php
new ExistingService();
```

## Coverage and boundaries

Level 0 provides the baseline symbol, class-model, invocation, and language checks. Coverage includes selected unknown classes, interfaces, functions, methods, properties, constants, attributes, imports, type references, visibility and declaration legality, missing required argument counts, and selected invalid language constructs. Extra positional arguments are enabled at level 1. `Level0.PropertyCallableType` reports native `callable` property types, including nullable, union, and promoted declarations, while PHPDoc callable annotations and native `Closure` types remain valid.

The implementation is intentionally partial compared with PHPStan level 0. Dynamic names, complete built-in and extension signatures, every modern syntax surface, level-1 magic-member diagnostics outside `$this`, and PHPDoc references are not fully covered. Higher-level type, variable-flow, dead-code, and deprecation checks are excluded when `analysis_level: 0` is selected.
