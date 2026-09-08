# PHPStan-compatible rules: level 3

<!-- rule-inventory: level=3 introduced=5 cumulative=23 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 5 registered levelled rules: `A.PROP.TYPE`, `A.RETURN.NEVER`, `A.RETURN.TYPE`, `A.RETURN.VOID`, and `Level3.ThrowType`.
- **Cumulative registered levelled rules:** 23. The assignment, binary-operation, void-purity, method, and PHPDoc rules from level 2 are cumulative and are not introduced here.
- **Checked-in differential pack:** 40 cases in `testdata/diagnostic-differential-level3`.

## Coverage and boundaries

Level 3 checks declared return types, values returned from `void` functions and methods, `never` return contracts, typed-property assignments, and selected throw targets. Return inference covers arithmetic, comparison, logical, unary-not, unary-numeric, and spaceship results in addition to the previously covered literal, call, property, cast, conditional, coalesce, and match expressions. Boolean literals retain `true`/`false` precision, and the built-in `fopen` signature is `resource|false`. Generic `$this` return inference prefers project-index `ResolveMethod` over unbound same-class AST signatures. Method-level `@template T` binds from callable arguments (`callable(): T` and unions such as `callable(...): T|CallbackInterface<T>`) using declared closure/arrow return types rather than inferred `Closure`; enclosing class templates also retain their identity in local return checks. Typed receivers without generic args use class `@extends` generic parents for method return inference (for example `$repo->find()` on a child repository, including a multi-hop `ServiceEntityRepository` parent). Unbound class templates inside generic arguments such as `EntityRepository<T>` stay as `T` when the current class still owns that template. Normally completing `finally` blocks preserve the selected try/catch outcome for return completeness. Return and property checks use immutable semantic facts and known class/property metadata where available.

Coverage remains partial for PHPDoc types, dynamic expressions, complex unions, every assignment operator, and the complete PHPStan type system. Levels 0 through 2 are cumulative.
