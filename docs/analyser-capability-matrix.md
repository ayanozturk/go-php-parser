# Analyser Capability Matrix

## Evidence contract

This matrix separates executable evidence from descriptive coverage claims. A capability is not described as PHPStan-compatible merely because a similarly named rule exists.

The checked-in differential packs live in `testdata/diagnostic-differential` (level 0), `testdata/diagnostic-differential-level1` (level 1), `testdata/diagnostic-differential-level2` (level 2), and `testdata/diagnostic-differential-level3` (level 3). Their manifests map each neutral PHP fixture to the exact diagnostic code expected from this engine and the exact PHPStan error identifier expected from the reference analyser. `cmd/diagnostic-diff` reports unexpected, missing, and duplicate diagnostics because it compares sorted identifier lists rather than only checking whether any error occurred.

Run the local engine gate:

```sh
go run ./cmd/diagnostic-diff --engine-only
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level1 --engine-only
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level2 --engine-only
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level3 --engine-only
```

Run the full differential against an installed PHPStan binary:

```sh
go run ./cmd/diagnostic-diff --phpstan-bin /absolute/path/to/phpstan --json
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level1 --phpstan-bin /absolute/path/to/phpstan --json
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level2 --phpstan-bin /absolute/path/to/phpstan --json
go run ./cmd/diagnostic-diff --fixtures testdata/diagnostic-differential-level3 --phpstan-bin /absolute/path/to/phpstan --json
```

The full report records the PHPStan version returned by the supplied executable. Results from different reference versions must not be merged without review. The 63-case level-0 pack, fourteen-case level-2 pack, and one-case level-3 pack were last fully verified against the pinned local reference `PHPStan 2.2.x-dev@e4ab62a`. The ordinary Go suite uses engine-only mode so it does not silently download or depend on an external analyser.

## Executable differential coverage

| Capability | Status | Fixture evidence | Engine diagnostic | PHPStan identifier | Current boundary |
| --- | --- | --- | --- | --- | --- |
| Unknown instantiated classes | Partial, differential-gated | `unknown-class` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a direct `new` of a missing class. Other class-not-found surfaces are gated separately below. |
| Unknown classes in static calls | Partial, differential-gated | `static-call-unknown-class` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves `Missing::method()`; dynamic class names remain outside the gate. |
| Unknown catch types | Partial, differential-gated | `unknown-catch-class` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a single `catch (MissingException $e)` clause. |
| Unknown parameter type references | Partial, differential-gated | `unknown-parameter-type` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a function parameter type; PHPDoc type references and other type contexts remain outside the gate. |
| Unknown function calls | Partial, differential-gated | `unknown-function` | `PHPStan.Level0.Symbols` | `function.notFound` | Built-in and extension-sensitive symbol coverage remains incomplete. |
| Unknown methods called on `$this` | Partial, differential-gated | `unknown-this-method` | `PHPStan.Level0.Symbols` | `method.notFound` | PHPStan level 0 covers `$this` only. Typed parameters and selected typed receiver expressions are gated at level 2; broader arbitrary-expression and union narrowing remain conservative. |
| Function argument counts | Partial, differential-gated | `argument-count` | `PHPStan.Level0.Invocation` | `arguments.count` | Covers a direct known function call; dynamic calls and constant-array unpacking remain outside the gate. |
| Method argument counts on `$this` | Partial, differential-gated | `this-method-argument-count` | `PHPStan.Level0.Invocation` | `arguments.count` | Covers a missing required argument on `$this`; named and unpacked calls remain outside the gate. |
| Private method visibility | Partial, differential-gated | `private-method-from-subclass` | `PHPStan.Level0.Invocation` | `method.private` | Proves a subclass `$this` call to a parent private method. Protected visibility begins at level 2 and is gated separately below. |
| Extending a final class | Partial, differential-gated | `extends-final-class` | `PHPStan.Level0.ClassModel` | `class.extendsFinal` | Proves a direct `extends` of a same-file final class. |
| Unknown implemented interfaces | Partial, differential-gated | `unknown-interface` | `PHPStan.Level0.ClassModel` | `interface.notFound` | Proves `implements MissingInterface`. Trait-use and interface-extends surfaces remain outside the gate. |
| Instantiating an abstract class | Partial, differential-gated | `instantiate-abstract-class` | `PHPStan.Level0.ClassModel` | `new.abstract` | Proves `new` of a same-file abstract class with no constructor arguments. |
| Instantiating an interface | Partial, differential-gated | `instantiate-interface` | `PHPStan.Level0.ClassModel` | `new.interface` | Proves interface instantiation; other class-like declaration legality checks remain outside the gate. |
| Overriding a final parent method | Partial, differential-gated | `final-method-override` | `PHPStan.Level0.ClassModel` | `method.parentMethodFinal` | Proves a same-file `final` method override. `@final` PHPDoc overrides remain outside the gate. |
| Final abstract classes | Partial, differential-gated | `final-abstract-class` | `PHPStan.Level0.ClassModel` | `phpstan.parse` | Proves the final-plus-abstract class parse surface; PHPStan reports a parse identifier rather than a class-model identifier. |
| Final abstract methods | Partial, differential-gated | `final-abstract-method` | `PHPStan.Level0.ClassModel` | `phpstan.parse` | Proves the final-plus-abstract method parse surface; PHPStan reports a parse identifier rather than a class-model identifier. |
| Overriding a final class constant | Partial, differential-gated | `final-constant-override` | `PHPStan.Level0.ClassModel` | `classConstant.final` | Proves a child declaration overriding a final inherited constant. Other constant compatibility checks remain outside the gate. |
| Private final class constants | Partial, differential-gated | `private-final-constant` | `PHPStan.Level0.ClassModel` | `classConstant.finalPrivate` | Proves the invalid private-plus-final constant modifier combination. Other constant modifier edge cases remain outside the gate. |
| Private interface constants | Partial, differential-gated | `interface-private-constant` | `PHPStan.Level0.ClassModel` | none | Proves an interface constant with invalid private visibility; PHPStan 2.2.5 is silent on this fixture at level 0. |
| Protected interface constants | Partial, differential-gated | `interface-protected-constant` | `PHPStan.Level0.ClassModel` | none | Proves an interface constant with invalid protected visibility; PHPStan 2.2.5 is silent on this fixture at level 0. |
| Consistent child constructors | Partial, differential-gated | `consistent-child-constructor` | `PHPStan.Level0.ClassModel` (twice) | `method.visibility`, `parameter.notOptional` | Proves visibility and required-parameter compatibility under `@phpstan-consistent-constructor`. Other signature variance remains outside the gate. |
| Private consistent constructors | Partial, differential-gated | `consistent-private-constructor` | `PHPStan.Level0.ClassModel` | `consistentConstructor.private` | Proves a non-final tagged class with a private constructor. Other tag and constructor interactions remain outside the gate. |
| Known class hierarchy without false positives | Partial, differential-gated | `clean-class-hierarchy` | none | none | Proves a public inherited `$this` call; not full false-positive parity across modifiers or visibility. |
| Unknown properties accessed on `$this` | Partial, differential-gated | `unknown-this-property` | `PHPStan.Level0.Symbols` | `property.notFound` | Proves `$this->missing`. Magic `__get` and other receiver forms remain outside the gate. |
| `$this` in a static method | Partial, differential-gated | `this-in-static` | `PHPStan.Level0.Symbols` (twice) | `variable.undefined` (twice) | Proves two static-method uses of `$this`; other undefined-variable flow remains in the level 1 pack. |
| Unknown class constants | Partial, differential-gated | `unknown-class-constant` | `PHPStan.Level0.Symbols` | `classConstant.notFound` | Proves `Known::MISSING` on a same-file class. |
| Unknown attribute classes | Partial, differential-gated | `unknown-attribute` | `PHPStan.Level0.Symbols` | `attribute.notFound` | Proves a top-level function attribute. Nested and parameter attributes remain outside the gate. |
| Unknown imported constants | Partial, differential-gated | `unknown-const-import` | `PHPStan.Level0.Symbols` | `constant.notFound` | Proves an imported constant reference whose declaration is missing. Other import kinds remain outside the gate. |
| Unknown property type references | Partial, differential-gated | `unknown-property-type` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a property type reference whose class cannot be resolved. PHPDoc property types remain outside the gate. |
| Unknown return type references | Partial, differential-gated | `unknown-return-type` | `PHPStan.Level0.Symbols` | `class.notFound` | Proves a return type reference whose class cannot be resolved. PHPDoc return types remain outside the gate. |
| Unknown static properties | Partial, differential-gated | `unknown-static-property` | `PHPStan.Level0.Symbols` | `staticProperty.notFound` | Proves an unknown static property on a known class. Dynamic class names and magic properties remain outside the gate. |
| Static access to an instance property | Partial, differential-gated | `static-access-instance-property` | `PHPStan.Level0.Symbols` | `property.staticAccess` | Proves a static access through a property declared as instance state. Other static/instance direction checks remain outside the gate. |
| Private class constant visibility | Partial, differential-gated | `private-class-constant` | `PHPStan.Level0.Symbols` | `classConstant.private` | Proves an access that violates private class-constant visibility. Inheritance and dynamic constant access remain outside the gate. |
| Protected class constant visibility | Partial, differential-gated | `protected-class-constant` | `PHPStan.Level0.Symbols` | `classConstant.protected` | Proves an access that violates protected class-constant visibility. Inheritance edge cases remain outside the gate. |
| Constructor argument counts | Partial, differential-gated | `constructor-argument-count` | `PHPStan.Level0.Invocation` | `arguments.count` | Covers a direct known constructor call with the wrong number of arguments. Dynamic calls and unpacking remain outside the gate. |
| Instantiating a protected constructor | Partial, differential-gated | `instantiate-protected-constructor` | `PHPStan.Level0.Invocation` | `new.protectedConstructor` | Proves `new` of a same-file protected constructor from an invalid context. Inheritance edge cases remain outside the gate. |
| Positional arguments after named arguments | Partial, differential-gated | `named-before-positional` | `PHPStan.Level0.Invocation` | `argument.missing`, `argument.positionalAfterNamed` | Proves the invalid ordering and PHPStan's skipped-argument diagnostic pair. Dynamic calls and unpacking remain outside the gate. |
| Instantiating a private constructor | Partial, differential-gated | `instantiate-private-constructor` | `PHPStan.Level0.Invocation` | `new.privateConstructor` | Proves `new` of a same-file private constructor. Inherited and dynamic construction contexts remain outside the pack. |
| Constructor arguments without a constructor | Partial, differential-gated | `extra-constructor-arguments` | `PHPStan.Level0.Invocation` | `new.noConstructor` | Proves `new NoCtor(1)`. Required constructor arity is a separate PHPStan identifier. |
| Unknown named arguments | Partial, differential-gated | `unknown-named-parameter` | `PHPStan.Level0.Invocation` | `argument.missing`, `argument.unknown` | PHPStan 2.2.5 also emits `argument.missing` for the skipped positional parameter; this engine reports only the unknown name. |
| `printf` argument counts | Partial, differential-gated | `printf-arguments` | `PHPStan.Level0.Invocation` | `argument.printf` | Proves a literal format string with too few values. Dynamic formats and broader format-string compatibility remain outside the gate. |
| Instance calls to static methods | Level-0 clean boundary | `instance-call-static-method` (level 0 pack) | none | none | The pinned reference is silent at level 0; the engine no longer reports the valid instance-call syntax as an invocation error. |
| Protected methods on known receivers | Differential-gated at levels 0 and 2 | `protected-method-known-receiver` (level 0 and level 2 packs) | none at level 0; `PHPStan.Level2.MethodVisibility` at level 2 | none at level 0; `method.protected` at level 2 | Level 0 remains clean; level 2 gates a protected call on a known receiver. Other receiver and inheritance contexts remain outside the gate. |
| Unknown methods on typed receivers | Differential-gated at level 2 | `unknown-method-typed-parameter`, `unknown-method-assigned-new`, `unknown-method-direct-new`, `unknown-method-return-chain`, `unknown-method-typed-property`, `unknown-method-named-function-chain`, `unknown-method-ternary-receiver`, `unknown-method-union-all-missing`, `unknown-method-intersection-all-missing` | `PHPStan.Level2.MethodExistence` | `method.notFound` | Covers typed parameters, variables assigned from `new`, direct `new` expressions, method and named-function return chains, typed property chains, class-only ternaries, and union/intersection receivers where every resolved class lacks the method. Variable calls, dynamic construction, nullable/non-object branches, and full DNF member availability remain conservative. |
| Known or conservative method receivers | Differential-gated at level 2 | `known-method-receiver`, `mixed-method-receiver`, `union-method-receiver`, `known-method-intersection-receiver` | none | none | Known methods, `mixed`, and union/intersection receivers where one member provides the method remain clean. Full DNF member availability remains conservative because the current type representation flattens union and intersection structure. |
| Unknown used traits | Partial, differential-gated | `unknown-trait` | `PHPStan.Level0.ClassModel` | `trait.notFound` | Proves `use MissingTrait`. |
| Instantiating an enum | Partial, differential-gated | `instantiate-enum` | `PHPStan.Level0.ClassModel` | `new.enum` | Proves enum instantiation; other enum and trait legality checks remain outside the gate. |
| Abstract methods in a concrete class | Partial, differential-gated | `abstract-method-in-concrete-class` | `PHPStan.Level0.ClassModel` | `method.abstract` (twice) | PHPStan 2.2.5 emits two `method.abstract` identifiers for the same method. |
| Missing interface method implementations | Partial, differential-gated | `missing-interface-method` | `PHPStan.Level0.ClassModel` | `method.abstract` | Proves a class that implements an interface without the required method. Multiple interfaces and trait interactions remain outside the gate. |
| Constructor return types | Partial, differential-gated | `constructor-return-type` | `PHPStan.Level0.ClassModel` | `constructor.returnType` | Proves `__construct(): void`. |
| Readonly class extending a non-readonly class | Partial, differential-gated | `readonly-extends-non-readonly` | `PHPStan.Level0.ClassModel` | `class.readOnly` | Proves one direction of readonly inheritance validation; other readonly restrictions remain outside the gate. |
| Non-readonly class extending a readonly class | Partial, differential-gated | `mutable-extends-readonly` | `PHPStan.Level0.ClassModel` | `class.nonReadOnly` | Proves the reverse readonly inheritance violation. Other readonly restrictions remain outside the gate. |
| Instantiating a trait | Partial, differential-gated | `instantiate-trait` | `PHPStan.Level0.ClassModel` | `new.trait` | Proves `new` of a trait declaration. Trait composition and other trait legality checks remain outside the gate. |
| Private abstract methods | Partial, differential-gated | `abstract-private-method` | `PHPStan.Level0.ClassModel` | `method.abstractPrivate` | Proves the invalid combination of private and abstract method modifiers. Other modifier combinations remain outside the gate. |
| Missing abstract parent method implementations | Partial, differential-gated | `missing-abstract-parent-method` | `PHPStan.Level0.ClassModel` | `method.abstract` | Proves a concrete child that leaves an abstract parent method unimplemented. Multiple-inheritance and trait interactions remain outside the gate. |
| Non-public interface methods | Partial, differential-gated | `interface-private-method` | `PHPStan.Level0.ClassModel` | `method.visibility` | Proves an interface method with invalid visibility. Interface method signatures and inherited visibility edge cases remain outside the gate. |
| Duplicate enum case values | Partial, differential-gated | `enum-duplicate-value` | `PHPStan.Level0.ClassModel` | `enum.duplicateValue` | Proves duplicate backed-enum case values. Other enum legality checks remain outside the gate. |
| Duplicate literal array keys | Partial, differential-gated | `duplicate-array-key` | `PHPStan.Level0.Language` | `array.duplicateKey` | Proves a string-key duplicate in an array literal. |
| Undefined goto labels | Partial, differential-gated | `unknown-goto` | `PHPStan.Level0.Language` | `goto.labelUndefined` | Proves `goto missing;` with no matching label. |
| Incrementing a literal | Partial, differential-gated | `increment-literal` | `PHPStan.Level0.Language` | `phpstan.parse` | Proves a non-writable increment target; PHPStan reports a parse identifier. |
| Missing included files | Partial, differential-gated | `missing-include` | `PHPStan.Level0.Language` | `include.fileNotFound` | Proves a literal include path that does not exist. Dynamic paths and other include/require resolution remain outside the gate. |
| Invalid regex patterns | Partial, differential-gated | `invalid-regex` | `PHPStan.Level0.Language` | `regexp.pattern` | Proves a statically invalid regular-expression pattern. Dynamic patterns and full PCRE compatibility remain outside the gate. |
| Invalid throw expressions | Differential-gated at levels 0 and 3 | `invalid-throw` (level 0 and level 3 packs) | none at level 0; `PHPStan.Level3.ThrowType` at level 3 | none at level 0; `throw.notThrowable` at level 3 | The level-0 fixture confirms the clean baseline; the level-3 pack gates resolved non-throwable classes. Unresolved and broader throw-type cases remain outside the gate. |
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
| Remaining class-model legality beyond the gated modifier, constructor, and interface-member cases | Partial | `PHPStan.Level0.ClassModel` |
| Remaining imports (class, function, and namespace forms) | Partial | `PHPStan.Level0.Symbols` |
| Static/instance call direction beyond the gated property case | Partial | `PHPStan.Level0.Invocation` |
| Remaining language checks (casts, include edge cases, and other non-gated checks) | Partial | `PHPStan.Level0.Language` |
| Return completeness, return types, and property assignment types | Partial, above level 0 | `A.RETURN.TYPE`, `A.PROP.TYPE` |
| Argument types | Partial, above levels 0–3 | `A.ARG.TYPE` |

## Unsupported milestone areas

| Capability | Status | Dependency |
| --- | --- | --- |
| Complete arbitrary-expression unknown method checks | Partial | Preserve union/intersection grouping for full DNF member availability, then extend facts to variable calls, dynamic construction, nullable/non-object branches, and remaining expression forms |
| PHPDoc validation parity | Not implemented | Complete PHPDoc type validation and source mapping |
| Full level 0 parity | Not implemented | Expand the differential pack across the agreed corpus and close reviewed mismatches |
| Quantified false-positive/false-negative thresholds | Not established | Larger reviewed differential corpus with pinned reference reports |

The broader descriptive inventory remains in `docs/phpstan-levels-0-3-rules-comparison.md`. That document must not be treated as executable parity evidence unless a row is linked to this differential pack.
