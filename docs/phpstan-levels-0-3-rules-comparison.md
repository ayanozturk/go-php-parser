# PHPStan Levels 0-3 vs Current Rule Coverage

## Scope

This document compares PHPStan rule levels 0, 1, 2, and 3 with the static-analysis rules currently implemented in this project.

Source for PHPStan level descriptions: [PHPStan Rule Levels](https://phpstan.org/user-guide/rule-levels). PHPStan levels are cumulative, so running level 3 includes checks from levels 0, 1, and 2.

This comparison is limited to analysis behavior. The project also implements PSR/formatting style rules, but those do not directly correspond to PHPStan's rule levels.

## Coverage Summary

| PHPStan level | PHPStan checks introduced at this level | Current project coverage |
| --- | --- | --- |
| 0 | Basic checks, unknown classes, unknown functions, unknown methods called on `$this`, wrong number of arguments passed to those methods and functions, always undefined variables | Partial, with active compatibility implementation behind `analysis_level: 0` |
| 1 | Possibly undefined variables, unknown magic methods and properties on classes with `__call` and `__get` | Not covered |
| 2 | Unknown methods checked on all expressions, PHPDoc validation | Not covered |
| 3 | Return types, types assigned to properties | Partial |

`analysis_level: 0` now runs a level-aware PHPStan compatibility rule set and suppresses current higher-level checks such as return type, property assignment type, argument type, and unreachable-code diagnostics. The implementation is grouped by behavior, not by PHPStan's individual rule classes.

## Detailed Comparison

| PHPStan level | PHPStan rule/check | Implemented? | Current rule code | Notes |
| --- | --- | --- | --- | --- |
| 0 | Basic semantic checks | Partial | `PHPStan.Level0.Language`, parser/command parse-error reporting | Covers selected language legality checks: duplicate literal array keys, undefined `goto` labels, literal include/require file existence, invalid `unset`/`void` casts, invalid increment/decrement targets, regex pattern validation, and printf/sprintf placeholder count checks. Full PHPStan basic-rule parity is not complete. |
| 0 | Unknown classes | Partial | `PHPStan.Level0.Symbols`, `PHPStan.Level0.ClassModel` | Covers unknown classes in `new`, `extends`, `implements`, interface `extends`, trait use, static calls, class constants/static properties, imports, type hints, catch types, and top-level attributes. Still missing some parser/AST surfaces and PHPStan guard/context suppressions. |
| 0 | Unknown functions | Partial | `PHPStan.Level0.Symbols` | Covers ordinary function calls and `use function`, backed by project and curated built-in function indexes. Built-in coverage is intentionally partial. |
| 0 | Unknown methods called on `$this` | Partial | `PHPStan.Level0.Symbols` | Covers direct `$this->method()` calls against the current class/project symbol index. Dynamic methods and PHPStan's full magic-method behavior are not covered. |
| 0 | Wrong number of arguments passed to methods and functions | Partial | `PHPStan.Level0.Invocation`; legacy `A.ARG.COUNT` outside explicit level mode | In `analysis_level: 0`, checks ordinary functions, constructors, static calls, `$this` method calls, named arguments, duplicate named arguments, positional-after-named, and unpack ordering for known signatures. Does not yet match PHPStan's full signature database or all dynamic/constant-array unpack cases. |
| 0 | Always undefined variables | Partial | `PHPStan.Level0.Variables` | Covers straightforward always-undefined variable reads, local params, assignment-created vars, foreach vars, catch vars, static vars, `$argc`/`$argv`, and `isset`/`empty` allowances. Branch analysis is intentionally coarse and does not yet match PHPStan's full scope engine. |
| 0 | Class/model legality | Partial | `PHPStan.Level0.ClassModel` | Covers duplicate class declarations, instantiating interface/trait/enum/abstract class, extending final/non-class/unknown classes, implementing non-interface/unknown interfaces, interface extends checks, trait-use validity, selected method visibility, static call to instance method, and selected property existence/staticness checks. Missing abstract method implementation completeness, constructor consistency, readonly inheritance details, and many modifier edge cases. |
| 0 | Type/reference legality | Partial | `PHPStan.Level0.Symbols`, `PHPStan.Level0.ClassModel` | Covers class-like type references in params, returns, properties, constants, interface methods, catches, imports, and top-level attributes. Does not yet cover every modern syntax location or PHPDoc references. |
| 1 | Possibly undefined variables | No | - | No control-flow-aware possibly-undefined-variable rule exists. |
| 1 | Unknown magic methods on classes with `__call` | No | - | No rule models `__call` as a PHPStan level 1 diagnostic. |
| 1 | Unknown magic properties on classes with `__get` | No | - | No rule models `__get` as a PHPStan level 1 diagnostic. |
| 2 | Unknown methods checked on all expressions | No | - | The current resolver supports some method lookup for other rules, but there is no diagnostic for unknown methods on arbitrary expression types. |
| 2 | PHPDoc validation | No | - | PHPDoc nodes/types exist in the AST layer, but there is no PHPDoc validation rule comparable to PHPStan level 2. |
| 3 | Return types | Partial | `A.RETURN.TYPE` | Checks declared return types against inferred return expression types for functions and methods. Coverage is narrower than PHPStan because inference and symbol knowledge are limited. |
| 3 | Types assigned to properties | Partial | `A.PROP.TYPE` | Checks assignments to typed properties when the property type can be resolved. Coverage is narrower than PHPStan because inference and cross-file symbol knowledge are limited. |

## Currently Implemented Analysis Rules

| Rule code | Description | Closest PHPStan level mapping |
| --- | --- | --- |
| `PHPStan.Level0.Symbols` | Groups PHPStan level-0 compatibility checks for unknown symbols, class-like references, imports, static calls, `$this` calls, constants, attributes, and selected property existence checks. | Partial PHPStan level 0 coverage. Enabled only when the selected analysis level includes 0. |
| `PHPStan.Level0.ClassModel` | Internal diagnostic code emitted by the level-0 rule group for class hierarchy/model legality. | Partial PHPStan level 0 coverage. |
| `PHPStan.Level0.Invocation` | Internal diagnostic code emitted by the level-0 rule group for argument-count and named-argument validity. | Partial PHPStan level 0 coverage. |
| `PHPStan.Level0.Variables` | Internal diagnostic code emitted by the level-0 rule group for always-undefined variable reads. | Partial PHPStan level 0 coverage. |
| `PHPStan.Level0.Language` | Internal diagnostic code emitted by the level-0 rule group for selected language legality checks. | Partial PHPStan level 0 coverage. |
| `A.ARG.COUNT` | Legacy non-level-aware argument-count rule for resolved method and constructor calls. | Historical partial PHPStan level 0 coverage; explicit `analysis_level: 0` uses `PHPStan.Level0.Invocation` instead. |
| `A.RETURN.TYPE` | Checks function/method return expressions against declared return types. | Partial PHPStan level 3 coverage. Registered above level 0 so it is suppressed for `analysis_level: 0`. |
| `A.PROP.TYPE` | Checks assigned values against resolved property types. | Partial PHPStan level 3 coverage. Registered above level 0 so it is suppressed for `analysis_level: 0`. |
| `A.ARG.TYPE` | Checks resolved method/constructor argument value types against declared parameter types. | Similar to PHPStan level 5, outside this level 0-3 comparison. Registered above level 0. |
| `Generic.CodeAnalysis.UnreachableCode` | Reports statements after terminating statements such as `return`, `throw`, `exit`, or `die`. | Similar to PHPStan level 4 dead-code checks, outside this level 0-3 comparison. Registered above level 0. |
| `Generic.CodeAnalysis.EmptyStatement` | Reports standalone empty statements and empty control-structure bodies. | PHPCS-style code-quality rule; no direct PHPStan level 0-3 mapping. |
| `Generic.CodeAnalysis.AssignmentInCondition` | Reports assignments inside conditions. | PHPCS-style code-quality rule; no direct PHPStan level 0-3 mapping. |
| `PSR1.Files.SideEffects` | Reports files that mix symbol declarations with side effects. | PSR-1/style rule; no direct PHPStan level 0-3 mapping. |

## Currently Implemented Style Rules

These rules are implemented in the project but are not PHPStan rule-level checks:

| Rule code |
| --- |
| `Generic.Arrays.DisallowLongArraySyntax` |
| `Generic.Formatting.DisallowMultipleStatements` |
| `Generic.Functions.FunctionCallArgumentSpacing` |
| `PSR1.Classes.ClassConstantName` |
| `PSR1.Classes.ClassDeclaration.PascalCase` |
| `PSR1.Classes.ClassInstantiation` |
| `PSR1.Methods.CamelCapsMethodName` |
| `PSR12.Classes.ClosingBraceOnOwnLine` |
| `PSR12.Classes.OpenBraceOnOwnLine` |
| `PSR12.ControlStructures.ControlStructureSpacing` |
| `PSR12.ControlStructures.ElseIfDeclaration` |
| `PSR12.Files.EndFileNewline` |
| `PSR12.Files.EndFileNoTrailingWhitespace` |
| `PSR12.Files.NoBlankLineAfterPHPOpeningTag` |
| `PSR12.Files.NoSpaceBeforeSemicolon` |
| `PSR12.Methods.VisibilityDeclared` |

## Implementation Gaps for PHPStan Level 0-3 Parity

To get closer to PHPStan levels 0-3, the next missing areas are:

1. Complete PHPStan 2.2.x level-0 rule parity across all registered rule classes, especially modifier legality, constructor consistency, missing abstract method implementations, readonly inheritance/property edge cases, class constant legality, enum-specific rules, throw expression validity, and PHPStan API restriction rules.
2. Complete namespace/use and parser coverage for all syntax locations: grouped use imports, function/const aliases, nested attributes, promoted-property attributes, anonymous classes, magic constants, declare placement/value checks, break/continue levels, property hooks, pipe operator, and newer PHP-version-gated syntax.
3. PHPStan-style reflection guards and context suppressions for `class_exists`, `interface_exists`, `trait_exists`, `enum_exists`, `function_exists`, `method_exists`, and `defined`.
4. A broader built-in function/class/constant/signature database, including extension-sensitive symbols and more precise constructor/function signatures.
5. More precise call handling: variadics, named args to variadics, unpacked constant arrays, no-constructor classes called with args, private/protected constructors, dynamic names with known constant-string values, and method visibility across inheritance.
6. More precise level-0 scope analysis: always undefined vs maybe undefined, branch intersection, by-reference writes, closure `use`, globals, compact variables, static variables, and `$this` in static/global contexts.
7. PHPStan level 1 possibly-undefined variable analysis and magic method/property diagnostics.
8. PHPStan level 2 arbitrary-expression method existence checks and PHPDoc validation.
9. Broader type inference and symbol resolution to make existing level-3 return/property checks closer to PHPStan behavior.

## Current Level-0 Compatibility Tests

Focused regression tests currently cover:

| Fixture/test area | Coverage |
| --- | --- |
| Unknown symbols and function arguments | Unknown functions, unknown instantiated classes, named-argument ordering, unknown named parameters. |
| Class model and `$this` methods | Extending final classes, implementing unknown interfaces, undefined `$this` methods. |
| Cross-file project index | Namespaced classes and functions resolved across files. |
| Duplicate declarations | Duplicate class declarations reported only for the file containing the duplicate declaration. |
| Type/use/catch/attribute references | Unknown imports, function imports, const imports, param/return/property type references, catch types, and top-level attributes. |
| Properties | Undefined `$this` properties, undefined static properties, and static access to instance properties. |
| Variables and language checks | Always undefined variable reads, `isset`/`empty` allowances, undefined labels, and duplicate array keys. |
| Level filtering | `analysis_level: 0` does not emit current higher-level return/property/argument type or unreachable-code diagnostics. |
