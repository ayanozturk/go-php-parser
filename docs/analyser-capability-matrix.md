# Analyser Capability Matrix

## Evidence contract

This matrix separates executable evidence from descriptive coverage claims. A capability is not described as PHPStan-compatible merely because a similarly named rule exists.

The checked-in differential packs live in `testdata/diagnostic-differential` (level 0) and `testdata/diagnostic-differential-level1` (level 1). Their manifests map each neutral PHP fixture to the exact diagnostic code expected from this engine and the exact PHPStan error identifier expected from the reference analyser. `cmd/diagnostic-diff` reports unexpected, missing, and duplicate diagnostics because it compares sorted identifier lists rather than only checking whether any error occurred.

Run the local engine gate:

```sh
go run ./cmd/diagnostic-diff --engine-only
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level1 --engine-only
```

Run the full differential against an installed PHPStan binary:

```sh
go run ./cmd/diagnostic-diff --phpstan-bin /absolute/path/to/phpstan --json
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level1 --phpstan-bin /absolute/path/to/phpstan --json
```

The full report records the PHPStan version returned by the supplied executable. Results from different reference versions must not be merged without review. The ordinary Go suite uses engine-only mode so it does not silently download or depend on an unpinned external analyser.

## Executable differential coverage

| Capability | Status | Fixture evidence | Engine diagnostic | PHPStan identifier | Current boundary |
| --- | --- | --- | --- | --- | --- |
| Unknown instantiated classes | Partial, differential-gated | `unknown-class` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a direct `new` of a missing class. Other class-not-found surfaces are gated separately below. |
| Unknown classes in static calls | Partial, differential-gated | `static-call-unknown-class` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves `Missing::method()`; dynamic class names remain outside the gate. |
| Unknown catch types | Partial, differential-gated | `unknown-catch-class` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a single `catch (MissingException $e)` clause. |
| Unknown parameter type references | Partial, differential-gated | `unknown-parameter-type` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a function parameter type; return, property, and PHPDoc type references remain outside the gate. |
| Unknown function calls | Partial, differential-gated | `unknown-function` | `PHPStan.Level0.Symbols` | `function.notFound` | Built-in and extension-sensitive symbol coverage remains incomplete. |
| Unknown methods called on `$this` | Partial, differential-gated | `unknown-this-method` | `PHPStan.Level0.Symbols` | `method.notFound` | PHPStan level 0 covers `$this` only. Arbitrary-expression method existence remains a level-2 gap. |
| Function argument counts | Partial, differential-gated | `argument-count` | `PHPStan.Level0.Invocation` | `arguments.count` | Covers a direct known function call; dynamic calls and constant-array unpacking remain outside the gate. |
| Method argument counts on `$this` | Partial, differential-gated | `this-method-argument-count` | `PHPStan.Level0.Invocation` | `arguments.count` | Covers a missing required argument on `$this`; named, unpacked, and constructor arity remain outside the gate. |
| Private method visibility | Partial, differential-gated | `private-method-from-subclass` | `PHPStan.Level0.Invocation` | `method.private` | Proves a subclass `$this` call to a parent private method. Protected calls on known receivers are reported by this engine at level 0 but PHPStan 2.2.5 is silent on that fixture at level 0, so they are not in this pack. |
| Extending a final class | Partial, differential-gated | `extends-final-class` | `PHPStan.Level0.ClassModel` | `class.extendsFinal` | Proves a direct `extends` of a same-file final class. |
| Unknown implemented interfaces | Partial, differential-gated | `unknown-interface` | `PHPStan.Level0.ClassModel` | `interface.notFound` | Proves `implements MissingInterface`. Trait-use and interface-extends surfaces remain outside the gate. |
| Instantiating an abstract class | Partial, differential-gated | `instantiate-abstract-class` | `PHPStan.Level0.ClassModel` | `new.abstract` | Proves `new` of a same-file abstract class with no constructor arguments. |
| Instantiating an interface | Partial, differential-gated | `instantiate-interface` | `PHPStan.Level0.ClassModel` | `new.interface` | Trait and enum instantiation remain outside the gate. |
| Overriding a final parent method | Partial, differential-gated | `final-method-override` | `PHPStan.Level0.ClassModel` | `method.parentMethodFinal` | Proves a same-file `final` method override. `@final` PHPDoc overrides remain outside the gate. |
| Known class hierarchy without false positives | Partial, differential-gated | `clean-class-hierarchy` | none | none | Proves a public inherited `$this` call; not full false-positive parity across modifiers or visibility. |
| Unknown properties accessed on `$this` | Partial, differential-gated | `unknown-this-property` | `PHPStan.Level0.Symbols` | `property.notFound` | Proves `$this->missing`. Static property existence and magic `__get` remain outside the gate. |
| Unknown class constants | Partial, differential-gated | `unknown-class-constant` | `PHPStan.Level0.Symbols` | `classConstant.notFound` | Proves `Known::MISSING` on a same-file class. |
| Unknown attribute classes | Partial, differential-gated | `unknown-attribute` | `PHPStan.Level0.Symbols` | `attribute.notFound` | Proves a top-level function attribute. Nested and parameter attributes remain outside the gate. |
| Instantiating a private constructor | Partial, differential-gated | `instantiate-private-constructor` | `PHPStan.Level0.Invocation` | `new.privateConstructor` | Proves `new` of a same-file private constructor. Protected constructors remain outside the pack. |
| Constructor arguments without a constructor | Partial, differential-gated | `extra-constructor-arguments` | `PHPStan.Level0.Invocation` | `new.noConstructor` | Proves `new NoCtor(1)`. Required constructor arity is a separate PHPStan identifier. |
| Unknown named arguments | Partial, differential-gated | `unknown-named-parameter` | `PHPStan.Level0.Invocation` | `argument.missing`, `argument.unknown` | PHPStan 2.2.5 also emits `argument.missing` for the skipped positional parameter; this engine reports only the unknown name. |
| Unknown used traits | Partial, differential-gated | `unknown-trait` | `PHPStan.Level0.ClassModel` | `trait.notFound` | Proves `use MissingTrait`. |
| Instantiating an enum | Partial, differential-gated | `instantiate-enum` | `PHPStan.Level0.ClassModel` | `new.enum` | Trait instantiation remains outside the gate. |
| Abstract methods in a concrete class | Partial, differential-gated | `abstract-method-in-concrete-class` | `PHPStan.Level0.ClassModel` | `method.abstract` (twice) | PHPStan 2.2.5 emits two `method.abstract` identifiers for the same method. |
| Missing interface method implementations | Partial, differential-gated | `missing-interface-method` | `PHPStan.Level0.ClassModel` | `method.abstract` | Proves a class that implements an interface without the required method. Abstract parent methods remain outside the gate. |
| Constructor return types | Partial, differential-gated | `constructor-return-type` | `PHPStan.Level0.ClassModel` | `constructor.returnType` | Proves `__construct(): void`. |
| Readonly class extending a non-readonly class | Partial, differential-gated | `readonly-extends-non-readonly` | `PHPStan.Level0.ClassModel` | `class.readOnly` | The reverse (mutable extending readonly) remains outside the gate. |
| Duplicate literal array keys | Partial, differential-gated | `duplicate-array-key` | `PHPStan.Level0.Language` | `array.duplicateKey` | Proves a string-key duplicate in an array literal. |
| Undefined goto labels | Partial, differential-gated | `unknown-goto` | `PHPStan.Level0.Language` | `goto.labelUndefined` | Proves `goto missing;` with no matching label. |
| Always undefined variables | Partial, differential-gated | `undefined-variable` (level 1 pack) | `PHPStan.Level1.Variables` | `variable.undefined` | PHPStan 2.2.5 introduces this diagnostic at level 1, including for an always-undefined top-level read. |
| Possibly undefined variables | Partial, differential-gated | `branch-defined-variable`, `while-defined-variable`, `closure-by-value-undefined-variable`, `reference-assignment`, `dynamic-receiver-by-reference`, `nested-continue-two-undefined-variable`, `switch-continue-two-undefined-variable`, `break-two-finally-undefined-variable`, `unknown-array-extract`, `builtin-reference-input-output-parameters` | `PHPStan.Level1.Variables` | `variable.undefined` | Joined facts cover conditionals, bounded loop convergence, `switch`, `try`/`catch`/`finally`, explicit closure captures, direct reference assignment, conservative dynamic-receiver arguments, numeric multi-level loop/switch transfers, unknown-array `extract()` effects, and selected input/output built-in references. Dynamic calls and dynamic transfer levels remain incomplete. |
| Exhaustive variable-flow controls | Partial, differential-gated | `exhaustive-branch-defined-variable`, `do-while-defined-variable`, `nested-break-two-defined-variable` | none | none | Proves selected clean branch, loop, and multi-level transfer controls, not full false-positive parity across all control flow. |
| Reference-defined variables | Partial, differential-gated | `closure-by-reference-definition`, `by-reference-output-parameter`, `instance-method-by-reference`, `static-method-by-reference`, `constructor-by-reference`, `builtin-reference-output-parameters` | none | none | Resolved function, `$this`, `self`/`parent`, explicit static, known `new` receiver, and constructor signatures preserve reference/output metadata. Core/standard built-ins cover regex match/count outputs, scanning variadics, process/status outputs, header locations, and common array/type input-output references. Dynamically typed receivers and extension-dependent signature metadata remain outside the definition gate. |
| Dynamic variable and `extract()` effects | Partial, differential-gated | `known-dynamic-read`, `dynamic-writes-do-not-define`, `direct-constant-extract`, `assigned-constant-extract`, `unknown-array-extract` | `PHPStan.Level1.Variables` or none | `variable.undefined` or none | Known-string dynamic reads resolve the target; dynamic writes remain non-defining to match PHPStan; direct and assigned constant-array keys become defined; unknown arrays make arbitrary names possible. Complex constant expressions, complete flag/prefix semantics, and broader symbol-table mutation remain incomplete. |
| Known symbols without false positives | Partial, differential-gated | `clean-known-symbols` | none | none | The clean fixture covers a declared function and compatible call, not the full false-positive surface. |

## Implemented but not yet differential-gated

These areas have repository unit coverage but no checked-in PHPStan differential fixture yet:

| Area | Current status | Principal diagnostic codes |
| --- | --- | --- |
| Remaining class-model legality (final+abstract parse cases, abstract-private/final methods, missing abstract parents, mutable-extends-readonly, trait instantiation, interface constant visibility) | Partial | `PHPStan.Level0.ClassModel` |
| Imports, return types, property types, and class-constant visibility | Partial | `PHPStan.Level0.Symbols` |
| Protected constructors, named-before-positional extras, and static/instance call direction | Partial | `PHPStan.Level0.Invocation` |
| Static property existence and static-access-to-instance | Partial | `PHPStan.Level0.Symbols` |
| Remaining language checks (includes, casts, regex, printf) | Partial | `PHPStan.Level0.Language` |
| Return completeness, return types, and property assignment types | Partial, above level 0 | `A.RETURN.TYPE`, `A.PROP.TYPE` |
| Argument types | Partial, above levels 0–3 | `A.ARG.TYPE` |

## Unsupported milestone areas

| Capability | Status | Dependency |
| --- | --- | --- |
| Arbitrary-expression unknown method checks | Not implemented | Reusable expression types and broader member resolution |
| PHPDoc validation parity | Not implemented | Complete PHPDoc type validation and source mapping |
| Full level 0 parity | Not implemented | Expand the differential pack across the agreed corpus and close reviewed mismatches |
| Quantified false-positive/false-negative thresholds | Not established | Larger reviewed differential corpus with pinned reference reports |

The broader descriptive inventory remains in `docs/phpstan-levels-0-3-rules-comparison.md`. That document must not be treated as executable parity evidence unless a row is linked to this differential pack.
