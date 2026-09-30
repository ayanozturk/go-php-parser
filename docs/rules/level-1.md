# PHPStan-compatible rules: level 1

<!-- rule-inventory: level=1 introduced=2 cumulative=4 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 2 registered levelled rules: `Level1.Variables` and `Level1.Core`.
- **Cumulative registered levelled rules:** 4 (including the two level-0 entries).
- **Checked-in differential pack:** 36 cases in `testdata/diagnostic-differential-level1`, pinned to PHPStan 2.2.5.

## Coverage

`Level1.Variables` reports always-undefined and possibly-undefined variable reads using joined flow facts. It covers branches, short-circuit expressions, ternaries, bounded loops, `foreach` values read after the loop, `switch`, `try`/`catch`/`finally`, interpolated strings, globals and statics, destructuring, closure captures, references, selected by-reference outputs, known-string dynamic reads, `extract`, `compact`, and `isset`/`empty` suppression.

`Level1.Core` adds these PHPStan level-1 checks:

- Undefined global constant reads (`constant.notFound`).
- Unused non-promoted constructor parameters and unused closure `use` captures.
- Redundant `isset()`, `empty()`, and `??` checks where the current parameter or PHPDoc type proves the result.

Existing level-0 rule entries also implement cumulative PHPStan behavior with level-dependent reporting:

- `Level0.Invocation` reports extra positional arguments to known functions and constructors starting at level 1; missing required arguments remain level 0. Instantiating a class without a constructor reports `new.noConstructor` at level 0 and above.
- `Level0.Symbols` reports unknown `$this` method/property accesses on classes declaring `__call` or `__get` starting at level 1; PHPStan suppresses those at level 0.

The differential fixtures include clean controls for declared constants and members, used constructor parameters, promoted constructor properties, used closure captures, nullable values, unknown boolean emptiness, and correctly sized calls.

## Boundaries

Variable-flow limitations remain for dynamic calls, dynamic transfer levels, complex dynamic-name expressions, and extension-dependent by-reference signatures. Redundancy checks currently use known function parameter types and do not reproduce all PHPStan expression-flow narrowing. Constant lookup depends on the project index and has incomplete namespace/import edge cases. The optional bleeding-edge `assign.byRefForeachExpr` rule is disabled by PHPStan's default feature-toggle configuration and is outside this default-level pack.

Level 1 includes the cumulative level-0 rule set. Run the reference comparison with:

```sh
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level1 --phpstan-bin /path/to/phpstan
```
