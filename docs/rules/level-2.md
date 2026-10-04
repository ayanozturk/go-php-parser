# PHPStan-compatible rules: level 2

<!-- rule-inventory: level=2 introduced=17 cumulative=21 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 17 registered levelled rules: `A.ASSIGN.OP.INVALID`, `A.BINARY.OP.INVALID`, `A.VOID.PURE`, `Level2.MethodExistence`, `Level2.MethodNonObject`, `Level2.MethodVisibility`, `Level2.PHPDocClass`, `Level2.PHPDocGenericLessTypes`, `Level2.PHPDocGenericMoreTypes`, `Level2.PHPDocMethodVariance`, `Level2.PHPDocNotGeneric`, `Level2.PHPDocGenericNotSubtype`, `Level2.PHPDocParamName`, `Level2.PHPDocParamType`, `Level2.PHPDocPropertyType`, `Level2.PHPDocReturnType`, and `Level2.PHPDocTemplateVariance`.
- **Cumulative registered levelled rules:** 21.
- **Checked-in differential pack:** 118 cases in `testdata/diagnostic-differential-level2`.

## Registered rules, one by one

### `A.ASSIGN.OP.INVALID`

**What it checks:** Checks selected compound assignments whose operands have incompatible types.

**Why it helps:** The assignment can fail at runtime or obscure an incorrect operator choice.

**Example that reports:**

```php
<?php
$count = 1;
$count += "x";
```

**A safer form:**

```php
<?php
$count = 1;
$count += 2;
```

### `A.BINARY.OP.INVALID`

**What it checks:** Checks selected binary operations with incompatible operand types.

**Why it helps:** Finding the mismatch early avoids runtime failures and clarifies the intended operation.

**Example that reports:**

```php
<?php
$result = [] + 4;
```

**A safer form:**

```php
<?php
$result = [] + [4];
```

### `A.VOID.PURE`

**What it checks:** See the pure-void check: reports selected void functions that only return and perform no work.

**Why it helps:** Removing a no-op wrapper reduces indirection and makes call behavior clearer.

**Example that reports:**

```php
<?php
function noop(): void { return; }
```

**A safer form:**

```php
<?php
// Remove noop() and its call sites.
```

### `Level2.MethodExistence`

**What it checks:** Checks selected method calls against known receiver classes and true-scope method_exists guards.

**Why it helps:** A misspelled or absent method is a common runtime error; checking known receiver types catches it before execution.

**Example that reports:**

```php
<?php
final class Worker {}
function run(Worker $worker): void { $worker->executeNow(); }
```

**A safer form:**

```php
<?php
final class Worker { public function executeNow(): void {} }
```

### `Level2.MethodNonObject`

**What it checks:** Reports selected method calls on receivers known to be non-objects.

**Why it helps:** Calling a method on a scalar or other non-object value fails at runtime and commonly reflects a missing type check.

**Example that reports:**

```php
<?php
function run(string $value): void { $value->save(); }
```

**A safer form:**

```php
<?php
function run(Repository $value): void { $value->save(); }
```

### `Level2.MethodVisibility`

**What it checks:** Checks selected calls that cannot access a known private or protected method from the current scope.

**Why it helps:** Visibility errors are deterministic runtime failures and can be corrected by using the public API or adjusting the scope deliberately.

**Example that reports:**

```php
<?php
class Secret { private function reveal(): void {} }
(new Secret())->reveal();
```

**A safer form:**

```php
<?php
class Secret { public function reveal(): void {} }
(new Secret())->reveal();
```

### `Level2.PHPDocClass`

**What it checks:** Resolves class references used in supported PHPDoc positions, including nested generic and callable types.

**Why it helps:** PHPDoc types guide analysis and editor tooling; a nonexistent class name makes those contracts misleading.

**Example that reports:**

```php
<?php
/** @param list<MissingRecord> $rows */
function render(array $rows): void {}
```

**A safer form:**

```php
<?php
/** @param list<Record> $rows */
function render(array $rows): void {}
```

### `Level2.PHPDocGenericLessTypes`

**What it checks:** Reports too few type arguments for supported generic classes that require template arguments.

**Why it helps:** Omitted type arguments erase useful element or key information and weaken checks downstream.

**Example that reports:**

```php
<?php
/** @param Collection $items */
function render(Collection $items): void {}
```

**A safer form:**

```php
<?php
/** @param Collection<int, Item> $items */
function render(Collection $items): void {}
```

### `Level2.PHPDocGenericMoreTypes`

**What it checks:** Reports too many type arguments supplied to supported generic classes.

**Why it helps:** Extra arguments usually mean the annotation does not match the class template contract and can corrupt inferred types.

**Example that reports:**

```php
<?php
/** @param Box<int, string> $box */
function useBox(Box $box): void {}
```

**A safer form:**

```php
<?php
/** @param Box<int> $box */
function useBox(Box $box): void {}
```

### `Level2.PHPDocGenericNotSubtype`

**What it checks:** Checks supported generic type arguments against declared template bounds, variance, and recorded inheritance substitutions.

**Why it helps:** A generic container with an incompatible type argument can violate the element contract expected by its methods.

**Example that reports:**

```php
<?php
/** @template T of Animal */
class Cage {}
/** @var Cage<string> $cage */
```

**A safer form:**

```php
<?php
/** @var Cage<Dog> $cage */
```

### `Level2.PHPDocMethodVariance`

**What it checks:** Checks supported method-level template variance declarations and rejects variance where callable template positions do not permit it.

**Why it helps:** Invalid variance claims can make otherwise incompatible generic APIs appear safe.

**Example that reports:**

```php
<?php
/** @template-covariant T */
function accept(callable(T): void $callback): void {}
```

**A safer form:**

```php
<?php
/** @template T */
function accept(callable(T): void $callback): void {}
```

### `Level2.PHPDocNotGeneric`

**What it checks:** Reports generic arguments applied to a class that has no matching template declaration.

**Why it helps:** An annotation that invents type parameters cannot describe the implementation and may hide a wrong class name.

**Example that reports:**

```php
<?php
/** @var DateTime<int> $date */
$date = new DateTime();
```

**A safer form:**

```php
<?php
/** @var DateTime $date */
$date = new DateTime();
```

### `Level2.PHPDocParamName`

**What it checks:** Checks supported @param tag names against the function or method’s declared parameters.

**Why it helps:** A misspelled parameter name leaves the intended contract unattached and can confuse tools and maintainers.

**Example that reports:**

```php
<?php
/** @param string $mesage */
function send(string $message): void {}
```

**A safer form:**

```php
<?php
/** @param string $message */
function send(string $message): void {}
```

### `Level2.PHPDocParamType`

**What it checks:** Compares supported @param annotations with native parameter declarations and validates referenced types.

**Why it helps:** Contradictory annotations make callers receive inconsistent guidance about accepted values.

**Example that reports:**

```php
<?php
/** @param int $id */
function find(string $id): void {}
```

**A safer form:**

```php
<?php
/** @param string $id */
function find(string $id): void {}
```

### `Level2.PHPDocPropertyType`

**What it checks:** Compares supported property PHPDoc types with native property types.

**Why it helps:** Property contracts are used across reads and writes; mismatches can create false confidence at every access site.

**Example that reports:**

```php
<?php
class Item {
    /** @var int */
    public string $code;
}
```

**A safer form:**

```php
<?php
class Item {
    /** @var string */
    public string $code;
}
```

### `Level2.PHPDocReturnType`

**What it checks:** Compares supported @return annotations with native return declarations and checks the documented type.

**Why it helps:** Callers depend on the return contract, so an annotation that disagrees with the declaration spreads incorrect assumptions.

**Example that reports:**

```php
<?php
/** @return int */
function title(): string { return "hello"; }
```

**A safer form:**

```php
<?php
/** @return string */
function title(): string { return "hello"; }
```

### `Level2.PHPDocTemplateVariance`

**What it checks:** Checks supported declaration-site template variance through nested generic and callable input/output positions.

**Why it helps:** Incorrect variance permits unsafe substitutions between generic types and can break consumers of the API.

**Example that reports:**

```php
<?php
/** @template-covariant T */
interface Sink { public function put(T $value): void; }
```

**A safer form:**

```php
<?php
/** @template T */
interface Sink { public function put(T $value): void; }
```

## Coverage and boundaries

Level 2 adds invalid assignment and binary-operator checks, PHPDoc validation, pure named `void` function detection, and method-call analysis for typed receivers. PHPDoc coverage checks simple and nested generic class references, including unknown classes in array-shape and callable signatures, excess or missing generic arguments on indexed template classes, generic arguments applied to non-generic classes, and template arguments against class bounds with variance-aware and inherited `@extends`/`@implements` substitutions. It checks declaration-site variance through nested generic parameters and callable input/output positions, rejects variance tags on callable templates, expands nested local `@phpstan-type`/`@psalm-type` aliases through imported class aliases in generic and callable signatures, checks tags for missing parameters, compares parameter, return, or property annotations with native declarations, and validates every branch of conditional return annotations against native unions. Alias expansion is token-safe, leaves quoted shape keys intact, and is bounded to 32 passes; parameterized aliases and `@phpstan-import-type` remain unsupported. Quoted literal atoms and integer-range bounds are not misclassified as classes. Closure signatures (`Closure(T): U` and `\Closure(T): U`) validate their nested types and compare their Closure base against native declarations, including imported and fully qualified Closure forms. All eleven PHPDoc rules reuse the existing structural traversal. Binary-operation coverage currently proves invalid numeric/string and array/scalar additions, with valid integer and array-union controls. `A.VOID.PURE` currently covers functions whose bodies contain only explicit returns; effectful and empty functions stay clean. Method checks report selected calls to undefined methods, methods on non-object values, and methods whose visibility is not accessible. True-scope `method_exists($var, 'name')` in `if`/`elseif`/`&&`/ternary treats the named method as provided for `Level2.MethodExistence`; dynamic `$obj->$method()` names remain outside the gate. The `known-method-negated-instanceof-or` clean control proves a known method on the right of `!$x instanceof T || …` after false-scope narrowing. The `known-method-phpdoc-mock-property` and `known-method-instanceof-keeps-mock` clean controls prove PHPUnit `@var T&MockObject` class properties and `instanceof T` on mock intersections.

Receiver inference remains deliberately conservative for dynamic calls, arbitrary unresolved expressions, and complex type combinations. Local non-parameterized PHPDoc aliases expand recursively through nested generic and callable types; imported class aliases in their bodies resolve through the file type context. Parameterized aliases, `@phpstan-import-type`, and cross-file type-alias imports remain unsupported. Alias expansion is bounded to 32 passes. Unresolvable signature bases and renamed Closure aliases remain outside this slice. Direct and nested class-template variance in callable parameter and return positions is covered for declared generic variance and callable signatures; property variance positions, call-site variance, and full tag legality remain partial. Generic bounds compare nested arguments using declaration-site variance and follow recorded generic inheritance substitutions. Template-bearing native-compatibility checks stay conservative. Numeric strings, bitwise operations, object conversions, and broader operator combinations remain outside the exact binary-operation gate. Levels 0 and 1 are cumulative.
