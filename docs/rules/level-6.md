# PHPStan-compatible rules: level 6

<!-- rule-inventory: level=6 introduced=5 cumulative=33 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 5 registered levelled rules: `Level6.MissingGenericType`, `Level6.MissingIterableValueType`, `Level6.MissingParameterType`, `Level6.MissingReturnType`, and `Level6.MissingPropertyType`.
- **Cumulative registered levelled rules:** 33.
- **Checked-in differential pack:** 30 cases in `testdata/diagnostic-differential-level6`.

## Registered rules, one by one

### `Level6.MissingGenericType`

**What it checks:** Reports selected generic class declarations or uses that omit required template arguments.

**Why it helps:** Explicit template arguments preserve the element or key type through the API instead of degrading to an unknown type.

**Example that reports:**

```php
<?php
/** @param Box $box */
function handle(Box $box): void {}
```

**A safer form:**

```php
<?php
/** @param Box<Item> $box */
function handle(Box $box): void {}
```

### `Level6.MissingIterableValueType`

**What it checks:** Reports supported array, iterable, and traversable declarations without enough value-type detail.

**Why it helps:** Knowing the item type allows checks on loops, offsets, and calls to catch mistakes inside the collection.

**Example that reports:**

```php
<?php
/** @param array $rows */
function render(array $rows): void {}
```

**A safer form:**

```php
<?php
/** @param array<int, Row> $rows */
function render(array $rows): void {}
```

### `Level6.MissingParameterType`

**What it checks:** Reports selected parameters that have neither a native type nor a usable PHPDoc type.

**Why it helps:** Parameter types document the boundary and let callers be checked before values enter the function.

**Example that reports:**

```php
<?php
function setName($name): void {}
```

**A safer form:**

```php
<?php
function setName(string $name): void {}
```

### `Level6.MissingPropertyType`

**What it checks:** Reports selected properties without a native or usable PHPDoc type.

**Why it helps:** A property type makes the class state contract visible and catches incompatible reads and writes.

**Example that reports:**

```php
<?php
class User { public $name; }
```

**A safer form:**

```php
<?php
class User { public string $name; }
```

### `Level6.MissingReturnType`

**What it checks:** Reports selected functions and methods without a native or usable PHPDoc return type.

**Why it helps:** Return types make the output contract explicit and allow callers to be checked.

**Example that reports:**

```php
<?php
function currentUser() { return new User(); }
```

**A safer form:**

```php
<?php
function currentUser(): User { return new User(); }
```

## Coverage and boundaries

Level 6 adds focused missing-type checks for generic class arguments, iterable value types, and untyped declarations. `Level6.MissingGenericType` reports non-iterable generic classes used without template arguments in selected parameter and return declarations. `Level6.MissingIterableValueType` reports `array` and `iterable` declarations without value types, including selected PHPDoc array forms, and bare iterable-generic classes in the Traversable lineage that omit template arguments. A bare Doctrine Collection control exercises this engine diagnostic against PHPStan's `missingType.generics` identifier; its correctly parameterized clean control checks both collection templates and the inherited iterator value type. Callable and Closure signatures recursively check parameter and return types for missing iterable detail. Multiline PHPDoc array shapes retain nested value types and suffixes. Enum methods retain their declaration PHPDoc, and inline `@var` annotations on promoted constructor parameters supply their iterable detail. For methods without local PHPDoc, precise iterable parameter and return types inherited from parent classes, interfaces, or traits satisfy the declaration; named and anonymous overrides inherit nested array shapes and `T[]` contracts without re-diagnosis, while explicit weak PHPDoc on inherited, trait, and anonymous-class contracts still reports, matching PHPStan. Class-level `@phpstan-type` aliases expand before missing-iterable checks. `Level6.MissingParameterType`, `Level6.MissingReturnType`, and `Level6.MissingPropertyType` report missing parameter, return, and property types while respecting PHPDoc declarations, explicit `mixed`, and constructor exemptions. The 30-case differential pack covers native and PHPDoc iterable failures, refined and inherited declarations, trait and anonymous-class failures and clean controls, Doctrine Collection contracts, multiline shapes, enum and promoted-parameter PHPDoc, PHPUnit data-provider iterable annotations, and clean mixed/void and PHPDoc controls; all fourteen failing cases remain silent at level 5. Broader missing-type inference for aliases, conditional types, dynamic or extension-provided types, and every iterable/generic declaration remains outside this exact gate. Selecting level 6 runs the cumulative rule set from levels 0 through 5.
