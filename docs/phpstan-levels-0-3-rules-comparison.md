# PHPStan Levels 0-3 vs Current Rule Coverage

## Scope

This document compares PHPStan rule levels 0, 1, 2, and 3 with the static-analysis rules currently implemented in this project.

Source for PHPStan level descriptions: [PHPStan Rule Levels](https://phpstan.org/user-guide/rule-levels). PHPStan levels are cumulative, so running level 3 includes checks from levels 0, 1, and 2.

This comparison is limited to analysis behavior. The project also implements PSR/formatting style rules, but those do not directly correspond to PHPStan's rule levels.

## Coverage Summary

| PHPStan level | PHPStan checks introduced at this level | Current project coverage |
| --- | --- | --- |
| 0 | Basic checks, unknown classes, unknown functions, unknown methods called on `$this`, wrong number of arguments passed to those methods and functions, always undefined variables | Partial |
| 1 | Possibly undefined variables, unknown magic methods and properties on classes with `__call` and `__get` | Not covered |
| 2 | Unknown methods checked on all expressions, PHPDoc validation | Not covered |
| 3 | Return types, types assigned to properties | Partial |

## Detailed Comparison

| PHPStan level | PHPStan rule/check | Implemented? | Current rule code | Notes |
| --- | --- | --- | --- | --- |
| 0 | Basic semantic checks | Partial | Parser/command parse-error reporting, not an analysis rule | The project parses PHP and can report parse failures, but there is no broad PHPStan-equivalent "basic checks" analysis rule. |
| 0 | Unknown classes | No | - | No rule currently validates class references against declarations, imports, autoload metadata, or resolver output as a reported diagnostic. |
| 0 | Unknown functions | No | - | Function calls are traversed by some rules, but no rule reports unknown function names. |
| 0 | Unknown methods called on `$this` | No | - | Method resolution is used by argument/type checks, but unresolved `$this->method()` calls are not reported as their own issue. |
| 0 | Wrong number of arguments passed to methods and functions | Partial | `A.ARG.COUNT` | Covers resolved method calls and constructor calls. It does not currently report wrong argument counts for ordinary function calls. |
| 0 | Always undefined variables | No | - | No definite undefined-variable rule exists. |
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
| `A.ARG.COUNT` | Checks resolved method and constructor argument counts. | Partial PHPStan level 0 coverage. |
| `A.RETURN.TYPE` | Checks function/method return expressions against declared return types. | Partial PHPStan level 3 coverage. |
| `A.PROP.TYPE` | Checks assigned values against resolved property types. | Partial PHPStan level 3 coverage. |
| `A.ARG.TYPE` | Checks resolved method/constructor argument value types against declared parameter types. | Similar to PHPStan level 5, outside this level 0-3 comparison. |
| `Generic.CodeAnalysis.UnreachableCode` | Reports statements after terminating statements such as `return`, `throw`, `exit`, or `die`. | Similar to PHPStan level 4 dead-code checks, outside this level 0-3 comparison. |
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

To get closer to PHPStan levels 0-3, the next missing rules are:

1. Unknown class detection.
2. Unknown function detection.
3. Unknown method detection for `$this`.
4. Unknown method detection for all expression types.
5. Undefined-variable analysis, split between always undefined and possibly undefined.
6. PHPDoc validation.
7. Function-call argument count checking for ordinary functions.
8. Broader type inference and symbol resolution to make return/property checks closer to PHPStan behavior.

