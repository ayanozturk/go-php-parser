# PHP 8+ Syntax Support Analysis - Missing Features

## Overview

This document analyzes the current PHP 8+ syntax support in the Go PHP Parser and identifies features that are missing or not fully implemented.

## Current Implementation Status

### ✅ Fully Implemented PHP 8.0 Features

- **Attributes**: `#[AttributeName]` - Lexer and parser support ✅
- **Union Types**: `TypeA|TypeB` - Full parser support ✅
- **Intersection Types**: `TypeA&TypeB` - Full parser support ✅
- **Constructor Property Promotion**: `public function __construct(private string $name)` ✅
- **Nullsafe Operator**: `$obj?->method()` - Lexer and token support ✅
- **Named Arguments**: `func(param: value)` - Token support exists ✅
- **Match Expressions**: Full parser support implemented ✅
- **Arrow Functions**: `fn($x) => $x * 2` - AST support ✅
- **Static Return Type**: `function foo(): static` ✅
- **Readonly Properties**: `readonly string $prop` ✅
- **Trailing Commas**: In parameter lists and arrays ✅

## Missing PHP 8.0 Features

### 1. Match Expressions
**Status**: ✅ **FULLY IMPLEMENTED**
- **Implementation**: Complete parser support added
- **Files Added/Modified**:
  - `parser/expression.go`: Added `T_MATCH` case to `parseSimpleExpression()`
  - `parser/match_parser.go`: New file with `parseMatchExpression()` and `parseMatchArm()` functions
  - `parser/match_test.go`: Comprehensive unit tests for match expression parsing
  - `parser/match_integration_test.go`: Integration tests for match expressions in various contexts
- **Features Supported**:
  - Basic match expressions: `match ($value) { 1 => 'one', 2 => 'two' }`
  - Multiple conditions per arm: `1, 2, 3 => 'small'`
  - Default arms: `default => 'other'`
  - Complex expressions in conditions and bodies
  - Trailing commas in match arms
  - Error handling for malformed expressions

**Example Code Now Supported:**
```php
$result = match ($value) {
    1 => 'one',
    2 => 'two',
    default => 'other'
};

// Multiple conditions
$size = match ($number) {
    1, 2, 3 => 'small',
    4, 5 => 'medium',
    default => 'large'
};

// Complex expressions
$result = match ($user->getRole()) {
    'admin' => $user->getAdminPanel(),
    'user' => 'Regular User',
    default => 'Guest'
};
```

### 2. First-class Callable Syntax
**Status**: ❌ **Not Implemented**
- **Issue**: No support for `$fn(...)` and `foo(...)` syntax
- **Evidence**:
  - No tokens defined for first-class callable syntax
  - No AST nodes for callable expressions
  - No parser implementation

**Example Code Not Supported:**
```php
$fn = strlen(...);
$result = $fn('hello');

$callback = foo(...);
```

### 3. Throw Expressions
**Status**: ❌ **Not Implemented**
- **Issue**: Cannot use `throw` as an expression
- **Evidence**:
  - `throw` is only supported as a statement, not expression
  - No AST node for throw expressions
  - No parser support for throw in expressions

**Example Code Not Supported:**
```php
$value = $condition ? 'valid' : throw new InvalidArgumentException();
```

## Missing PHP 8.1+ Features

### PHP 8.1 Features
**Status**: ❌ **Not Implemented**

1. **Enums with Backed Cases**
   ```php
   enum Status: string {
       case PENDING = 'pending';
       case APPROVED = 'approved';
   }
   ```

2. **Readonly Properties** (partially implemented - basic syntax only)
   - Missing: readonly classes, readonly in interfaces

3. **First-class Callable Syntax** (see above)

4. **New in Initializer** (pure intersection types)
   ```php
   class Foo {
       public function __construct(
           public (A&B) $prop
       ) {}
   }
   ```

5. **Never Return Type**
   ```php
   function redirect(): never {
       header('Location: /');
       exit();
   }
   ```

### PHP 8.2 Features
**Status**: ❌ **Not Implemented**

1. **Readonly Classes**
   ```php
   readonly class Point {
       public function __construct(
           public int $x,
           public int $y,
       ) {}
   }
   ```

2. **Disjunctive Normal Form (DNF) Types**
   ```php
   function foo((A&B)|C $param): (A&B)|C {}
   ```

3. **Constants in Traits**
   ```php
   trait WithConstants {
       const string NAME = 'value';
   }
   ```

4. **True, False, and Null Standalone Types**
   ```php
   function alwaysTrue(): true {}
   function alwaysFalse(): false {}
   function alwaysNull(): null {}
   ```

### PHP 8.3 Features
**Status**: ❌ **Not Implemented**

1. **Typed Class Constants**
   ```php
   class Foo {
       const string BAR = 'baz';
   }
   ```

2. **Dynamic Class Constant Fetch**
   ```php
   $class = 'Foo';
   $const = 'BAR';
   $value = $class::$const;
   ```

3. **Override Attribute**
   ```php
   class Child extends Parent {
       #[Override]
       public function method(): void {}
   }
   ```

4. **Deep-clone of readonly properties**
   - Complex interaction with readonly properties

## Parser Implementation Gaps

### 1. Expression Parsing
- **Issue**: `parseSimpleExpression()` doesn't handle `match` keyword
- **Location**: `parser/expression.go:199`
- **Missing**: Case for `token.T_MATCH` in switch statement

### 2. Statement Parsing
- **Issue**: No match expression parsing in statement context
- **Location**: `parser/statement.go`
- **Missing**: Match expression parsing logic

### 3. Token Recognition
- **Issue**: Some PHP 8+ keywords may not be fully recognized
- **Location**: `lexer/lexer_keywords.go`
- **Status**: Most PHP 8 keywords present, but may need verification

## Testing Coverage Gaps

### 1. Integration Tests
- **Issue**: No end-to-end parsing tests for complex PHP 8 expressions
- **Evidence**: Unit tests exist for AST nodes but no parser integration tests
- **Missing**: Tests parsing real `match` expressions from PHP code

### 2. Real-world Codebases
- **Issue**: Limited testing against real PHP 8+ codebases
- **Evidence**: Debug files contain some PHP 8 syntax but limited scope

## Priority Recommendations

### High Priority (Critical for PHP 8.0 Support)
1. **Throw Expressions** - Important language feature
2. **First-class Callable Syntax** - Advanced but commonly used

### Medium Priority (PHP 8.1+ Support)
1. **Readonly Classes** - Major PHP 8.2 feature
2. **DNF Types** - Complex type system feature
3. **Typed Class Constants** - PHP 8.3 feature

### Low Priority (Edge Cases)
1. **Dynamic Class Constant Fetch** - Advanced feature
2. **Override Attribute** - Optional attribute
3. **Deep-clone readonly properties** - Complex edge case

## Implementation Notes

### Match Expression Implementation
To implement match expressions:

1. Add `token.T_MATCH` case in `parseSimpleExpression()`
2. Create `parseMatchExpression()` function
3. Parse condition: `match (condition)`
4. Parse arms: `value => expression`
5. Handle default arm: `default => expression`

### Testing Strategy
1. Add parser integration tests for match expressions
2. Test against real PHP 8+ codebases
3. Add regression tests for edge cases
4. Verify compatibility with existing PHP 8 features

## Conclusion

The Go PHP Parser has solid foundation support for many PHP 8.0 features, but **match expressions** represent a critical gap that prevents claiming full PHP 8.0 support. Additional PHP 8.1+ features are also missing, particularly readonly classes and DNF types.

Priority should be given to implementing match expressions first, followed by throw expressions and first-class callable syntax to achieve complete PHP 8.0 compliance.
