# PHP 8+ Syntax Support Roadmap

This document outlines the major milestones and features required to achieve robust support for PHP 8+ syntax in the Go PHP parser.

---

## 1. Lexer Enhancements
- [ ] Support all PHP 8+ tokens: attributes (`#[...]`), match, nullsafe operator (`?->`), named arguments, promoted properties, etc.
- [ ] Recognize PHP 8+ keywords and contextual keywords.
- [ ] Support for PHPDoc and inline comments as tokens (for docblock parsing or skipping).

## 2. Parser: Core Syntax
- [ ] **Attributes**: Parse and attach attributes to functions, classes, properties, and parameters.
- [ ] **Constructor Property Promotion**: Support `public|protected|private [readonly] type $name` in constructor parameters.
- [ ] **Named Arguments**: Parse named arguments in function/method calls.
- [ ] **Union Types**: Parse union types (`TypeA|TypeB`) everywhere type hints are allowed.
- [ ] **Intersection Types**: Parse intersection types (`TypeA&TypeB`).
- [ ] **Nullable Types**: Parse `?Type` for all type hint positions.
- [ ] **Static Return Type**: Support `static` as a return type.
- [ ] **Trailing Commas**: Allow trailing commas in parameter and argument lists, arrays, etc.
- [ ] **Match Expression**: Parse `match` expressions.
- [ ] **Nullsafe Operator**: Parse `$foo?->bar()`.
- [ ] **Named Parameters in Calls**: Parse and represent named arguments in calls.

## 3. Parser: Advanced Features
- [ ] **Readonly Properties**: Parse `readonly` properties and parameters.
- [ ] **Enums**: Parse `enum` declarations and cases.
- [ ] **Fibers**: Parse `Fiber`-related syntax if needed.
- [ ] **Throw Expressions**: Support `throw` as an expression.
- [ ] **New Function Signatures**: Support mixed, never, and other new types.
- [ ] **Interface Inheritance**: Parse `interface Foo extends Bar`.
- [ ] **Class Constants Visibility**: Support `public|protected|private const`.
- [ ] **First-class Callable Syntax**: Parse `$fn(...)` and `foo(...)`.
- [ ] **Static Variables in Functions**: Parse `static $x = ...` inside functions.

## 4. PHPDoc and Comments
- [ ] Robustly skip or parse PHPDoc blocks inside interfaces, classes, and functions.
- [ ] Optionally, parse PHPDoc for type and param extraction.

## 5. Error Recovery & Reporting
- [ ] Improve parser error messages for incomplete or invalid PHP 8+ constructs.
- [ ] Implement error recovery strategies to continue parsing after non-fatal errors.

## 6. Testing & Fixtures
- [ ] Add comprehensive tests for each PHP 8+ feature.
- [ ] Use real-world open source PHP 8+ codebases as test fixtures.
- [ ] Add regression tests for previously reported parsing errors.

## 7. Performance & Parallelism
- [ ] Ensure new features do not degrade performance.
- [ ] Test and optimize parallel file scanning and parsing.

---

### Prioritization

1. **Lexer and Core Syntax**: Foundation for all other features.
2. **Advanced Features**: Add incrementally after core is robust.
3. **PHPDoc/Comments & Error Recovery**: For usability and real-world compatibility.
4. **Testing & Performance**: Continuous throughout development.

---

### Contribution Guidelines

- Open issues for each feature or bug.
- Submit tests with each PR.
- Reference this roadmap in PRs and issues.

---

### Status

- [ ] Full PHP 8+ support: _In Progress_
