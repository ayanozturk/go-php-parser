# PHPStan-compatible rules: level 3

<!-- rule-inventory: level=3 introduced=5 cumulative=26 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 5 registered levelled rules: `A.PROP.TYPE`, `A.RETURN.NEVER`, `A.RETURN.TYPE`, `A.RETURN.VOID`, and `Level3.ThrowType`.
- **Cumulative registered levelled rules:** 26. The assignment, binary-operation, void-purity, method, and PHPDoc rules from level 2 are cumulative and are not introduced here.
- **Checked-in differential pack:** 44 cases in `testdata/diagnostic-differential-level3`.

## Registered rules, one by one

### `A.PROP.TYPE`

**What it checks:** Checks assignments to typed properties against the property’s declared type.

**Why it helps:** Typed properties are runtime-enforced contracts; incompatible assignments fail when executed.

**Example that reports:**

```php
<?php
class Cart { public int $count; }
$cart = new Cart();
$cart->count = "many";
```

**A safer form:**

```php
<?php
$cart->count = 3;
```

### `A.RETURN.NEVER`

**What it checks:** Checks selected never-returning functions and methods for paths that return normally instead of terminating or throwing.

**Why it helps:** A never contract tells callers execution cannot continue; violating it makes control-flow reasoning unsafe.

**Example that reports:**

```php
<?php
function stop(): never { return; }
```

**A safer form:**

```php
<?php
function stop(): never { throw new RuntimeException(); }
```

### `A.RETURN.TYPE`

**What it checks:** Compares returned expressions with declared return types for supported inferred expressions.

**Why it helps:** Return mismatches violate the function contract and can cause downstream type errors.

**Example that reports:**

```php
<?php
function countItems(): int { return "three"; }
```

**A safer form:**

```php
<?php
function countItems(): int { return 3; }
```

### `A.RETURN.VOID`

**What it checks:** Reports returned values from functions or methods declared void.

**Why it helps:** A void declaration promises no value; returning one makes the API contract inconsistent.

**Example that reports:**

```php
<?php
function logMessage(): void { return "done"; }
```

**A safer form:**

```php
<?php
function logMessage(): void { echo "done"; return; }
```

### `Level3.ThrowType`

**What it checks:** Checks selected throw expressions to ensure the thrown value is a Throwable.

**Why it helps:** Throwing a non-exception value is invalid PHP and fails at runtime.

**Example that reports:**

```php
<?php
throw "failed";
```

**A safer form:**

```php
<?php
throw new RuntimeException("failed");
```

## Coverage and boundaries

Level 3 checks declared return types, values returned from `void` functions and methods, `never` return contracts, typed-property assignments, and selected throw targets. Return inference covers arithmetic, comparison, logical, unary-not, unary-numeric, and spaceship results in addition to the previously covered literal, call, property, cast, conditional, coalesce, and match expressions. Boolean literals retain `true`/`false` precision, and the built-in `fopen` signature is `resource|false`. Generic `$this` return inference prefers project-index `ResolveMethod` over unbound same-class AST signatures. Method-level `@template T` binds from callable arguments (`callable(): T` and unions such as `callable(...): T|CallbackInterface<T>`) using declared closure/arrow return types rather than inferred `Closure`; enclosing class templates also retain their identity in local return checks. Typed receivers without generic args use class `@extends` generic parents for method return inference (for example `$repo->find()` on a child repository, including a multi-hop `ServiceEntityRepository` parent). Unbound class templates inside generic arguments such as `EntityRepository<T>` stay as `T` when the current class still owns that template. Normally completing `finally` blocks preserve the selected try/catch outcome for return completeness. Return and property checks use immutable semantic facts and known class/property metadata where available.

Coverage remains partial for PHPDoc types, dynamic expressions, complex unions, every assignment operator, and the complete PHPStan type system. Levels 0 through 2 are cumulative.
