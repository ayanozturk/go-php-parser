# PHPStan-compatible rules: level 5

<!-- rule-inventory: level=5 introduced=1 cumulative=28 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `A.ARG.TYPE`.
- **Cumulative registered levelled rules:** 28.
- **Checked-in differential pack:** 103 cases in `testdata/diagnostic-differential-level5`.

## Registered rules, one by one

### `A.ARG.TYPE`

**What it checks:** Checks selected calls to known functions, methods, and constructors against parameter types, including supported PHPDoc and inferred types.

**Why it helps:** Passing the wrong value can cause runtime type errors or incorrect library behavior; checking the call site localizes the correction.

**Example that reports:**

```php
<?php
function save(int $id): void {}
save("abc");
```

**A safer form:**

```php
<?php
function save(int $id): void {}
save(42);
```

## Coverage and boundaries

`A.ARG.TYPE` reports incompatible arguments passed to resolved named functions, methods, and constructors. The 103-case pack covers positional, named, method, constructor, `self`-parameter, PHPDoc alias/list, generic function parameter bounds, unbound class/method-template parameters and bounds, closure-filter, Symfony form and dependency-injection return, Laravel Eloquent builder, string-narrowing, scalar/nested array-shape offsets, indexed PHPDoc shape returns, generic array value offsets, `key-of`/`value-of` shape and generic array projections, callable parameter and return signatures, nested PHPDoc shape offsets, optional offsets, and union-shape projections, native/PHPDoc refinement (including refinements flowing through function parameters that also declare native types), parameter-dependent function and method return branches, strict exact-string equality branches, and valid literal branch joins; failing cases match PHPStan's `argument.type` identifier at level 5. Failing cases remain silent at level 4. Nullable-only argument mismatches are gated at level 8, while an extra `false` union arm is gated at level 7, matching the reference analyser's thresholds. Method `self`/`static`/`parent` parameter types bind to the callee's declaring class, called class, and parent rather than the caller. Direct `list<T>` parameters and class-level `@phpstan-type` / `@psalm-type` list aliases accept compatible arrays rather than being resolved as namespaced classes. Unbound method/class templates with bounds expand to those bounds instead of fake classes unless a call-site argument binds them. Interface method `@template T` stays a template instead of a namespaced fake class. Property PHPDoc generics such as `InputBag<string>` refine native property types, so `query->get('search', '')` infers `string`. Arrow functions and closures are `Closure`, and `callable` accepts `Closure`. A local `@var` on the next expression statement asserts that variable's type, including after a nullable `getUser()` assignment. Conditional PHPDoc returns use a branch selected from a statically known argument; unresolved argument types retain the branch union. Conditional annotations continue to fall back to the native return when their tested class types are unresolved. A terminating `if (!is_string($a) || !is_string($b)) { return; }` narrows both variables to `string` afterwards. `is_string($x) && $x !== '' ? (int)$x : null` is a ternary of `int|null`, not a boolean `&&`. Symfony container `get(class-string<T>)` and Doctrine repository `find(class-string<T>)` bind the service/entity type and preserve nullable return behavior. Scalar and nested shape offsets preserve declared builtin types, including unions of quoted PHPDoc string values. Unbound template parameters are treated as mixed (or their declared bound) when no concrete type argument is present, avoiding a fabricated class mismatch. A truthy `$x?->prop` (or `$x->prop`) in an `if` narrows `$x` to non-null inside that branch. A non-terminating `if (!$x) { $x = new T; }` or `if ($x === false) { $x = 0; }` joins the assignment with the implicit else so later statements see the filled-in type. `$this->prop instanceof T` and `$obj->prop instanceof T` narrow that property in the then-branch so later arguments see `T`. A `while ($x !== null)` loop header narrows `$x` to non-null inside the loop body. Static methods, dynamic callables, unpacked arguments, extension-provided signatures, `@var` on statements other than expressions and returns, conditions beyond the supported type tests, structured function-template binding and structured method-template binding, Symfony's richer conditional service-id return expression, and the full PHPStan type lattice remain partial. The rule uses the existing shared argument-call traversal, so moving it from level 10 does not add a pass to the default analysis path.

Named generic functions bind declared direct template parameters from positional,
named, and variadic arguments, joining repeated contributions. A parameter of
`callable(): Result` also binds `Result` from a declared or supported inferred closure/arrow return or
known callable-variable return. The substituted function return feeds argument
checks, so `acceptInt(preserve('wrong'))` reports while preserving an integer is
clean. Template names need not start with `T`. Unresolved templates fall back to
their declared bound or `mixed`; unpacked and structured parameter inference,
dependent template bounds, and complex callback bodies remain partial.

Known standalone `callable(...)` / `Closure(...)` parameter contracts check
callback parameter contravariance, return covariance, and required, optional,
and variadic arity at level 5. Relative callback types bind to the callee, and
inherited generic signatures retain substituted callback metadata. Supported
unannotated arrows and simple closure bodies use parameter scopes, implicit
arrow captures, explicit closure captures, and isolated local assignments.
Declared return types remain authoritative. Loops, try/switch bodies,
generators, reference parameters, unresolved templates, and nesting beyond 16
remain conservative; nullable or union callback contracts and the complete
PHPStan callable/array type lattice are not signature-checked.
