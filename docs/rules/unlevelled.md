# Unlevelled analysis rules

<!-- rule-inventory: unlevelled=4 levelled=36 total=40 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

These rules are registered without a PHPStan-compatible analysis level:

- `A.ARG.COUNT`
- `Generic.CodeAnalysis.AssignmentInCondition`
- `Generic.CodeAnalysis.EmptyStatement`
- `PSR1.Files.SideEffects`

- **Introduced outside an exact level:** 4 registered unlevelled rules.
- **Cumulative registered levelled rules:** 36 (unchanged; these rules are not part of that total).
- **Total registered analysis rules:** 40, consisting of 36 levelled rules and these 4 unlevelled rules.
- **Checked-in differential pack:** None currently checked in.

The four unlevelled rules run only when `analysis_level` is unset. When an explicit analysis level is selected, only registered rules at or below that level are enabled.

## Registered rules, one by one

### `A.ARG.COUNT`

**What it checks:** Checks selected resolved method and constructor calls for argument-count mismatches in legacy unlevelled mode.

**Why it helps:** Missing or extra arguments often signal a caller that no longer matches the method contract.

**Example that reports:**

```php
<?php
function greet(string $name): void {}
greet();
```

**A safer form:**

```php
<?php
function greet(string $name): void {}
greet("Ada");
```

### `Generic.CodeAnalysis.AssignmentInCondition`

**What it checks:** Reports assignment expressions used directly as conditions in supported contexts.

**Why it helps:** A single equals sign may be accidental where a comparison was intended and can change branching behavior.

**Example that reports:**

```php
<?php
if ($ready = loadState()) { run(); }
```

**A safer form:**

```php
<?php
if ($ready === loadState()) { run(); }
```

### `Generic.CodeAnalysis.EmptyStatement`

**What it checks:** Reports selected empty statements that do no work, such as a stray semicolon after a control statement.

**Why it helps:** An accidental empty body can make a condition or loop silently do nothing.

**Example that reports:**

```php
<?php
if ($ready);
    run();
```

**A safer form:**

```php
<?php
if ($ready) { run(); }
```

### `PSR1.Files.SideEffects`

**What it checks:** Checks file-level executable side effects under the supported PSR-1 file-organization rule.

**Why it helps:** Keeping declarations separate from top-level work makes files predictable to include and safer to load.

**Example that reports:**

```php
<?php
$booted = startApplication();
function helper(): void {}
```

**A safer form:**

```php
<?php
function helper(): void {}
```

## Coverage and boundaries

`A.ARG.COUNT` is the legacy argument-count rule for resolved method and constructor calls. The level-aware level-0 invocation checks are used instead when explicit level mode is selected.

`Generic.CodeAnalysis.AssignmentInCondition` reports assignments used as conditions, `Generic.CodeAnalysis.EmptyStatement` reports selected empty statements, and `PSR1.Files.SideEffects` reports file-level side effects under the existing style-rule behavior. These rules are retained for compatibility and are not counted in any PHPStan level's cumulative levelled total.
