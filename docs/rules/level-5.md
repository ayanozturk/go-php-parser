# PHPStan-compatible rules: level 5

<!-- rule-inventory: level=5 introduced=1 cumulative=25 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `A.ARG.TYPE`.
- **Cumulative registered levelled rules:** 25.
- **Checked-in differential pack:** 16 cases in `testdata/diagnostic-differential-level5`.

## Coverage and boundaries

`A.ARG.TYPE` reports incompatible arguments passed to resolved named functions, methods, and constructors. The sixteen-case differential pack covers positional and named mismatches plus compatible scalar, subtype, PHP 8.3+ `DateTime::modify()` return, `self` parameter, generic Collection, unbound template-return, `@phpstan-type` list-alias, InputBag query `get`, and Closure-filter controls; all seven failing cases are silent at level 4 and match PHPStan's `argument.type` identifier at level 5. Method `self`/`static`/`parent` parameter types bind to the callee's declaring class, called class, and parent rather than the caller. Generic class names compare by the class after erasing `<...>` arguments, so `ArrayCollection` satisfies `Collection<string, User>`. Unbound method/class templates with bounds expand to those bounds instead of fake classes unless a call-site argument binds them. Property PHPDoc generics such as `InputBag<string>` refine native property types, so `query->get('search', '')` infers `string`. Class-level `@phpstan-type` and `@psalm-type` aliases expand to their definitions, so a `list<...>` alias accepts an array. Arrow functions and closures are `Closure`, and `callable` accepts `Closure`. Function/method-local PHPDoc template parameters that are not bare template names remain conservative `mixed` at call sites. Static methods, dynamic callables, unpacked arguments, extension-provided signatures, assignment `@var`, call-site template binding beyond `T*` names, and the full PHPStan type lattice remain partial. The rule uses the existing shared argument-call traversal, so moving it from level 10 does not add a pass to the default analysis path.
