# PHP Strom integration review

This document records the August 2026 review of `go-php-parser` as the parser and analysis engine used by the sibling `vscode-php-strom` project.

## Current relationship

PHP Strom's production indexer, semantic cache, diagnostics, and several language providers import this module under its current `go-phpcs` module name. PHP Strom also contains a separate parser under `server/parser`, but that implementation is used only by its `cmd/parse-test` utility and is not the parser used by editor features.

Both repositories' ordinary Go test suites passed during the review. The parser repository also passed `go vet ./...` and `go test -race ./...`. PHP Strom's race suite exposed a test-harness race in `TestDidChangeDebouncesOnTypeAnalysis`, where a `bytes.Buffer` is read while an asynchronous diagnostic notification writes to it. Its TypeScript compilation passed, while linting failed because ESLint 9 could not find an `eslint.config.*` file.

## Recommended changes

### 1. Always collect parser errors

Previously, `Parser.addError` appended errors only when the parser was constructed with debug mode enabled. PHP Strom, the compatibility metrics command, and the normal concurrent scanner all construct the parser with debug mode disabled. As a result, syntax errors were silently omitted and PHP Strom's syntax-diagnostic path received an empty error list.

Parser errors must be collected independently of presentation or logging verbosity. Debug mode should control diagnostic logging only, not parser correctness.

Status: implemented in this repository. The existing `Errors() []string` API is preserved so PHP Strom remains source-compatible. A follow-up should introduce structured parser errors containing a stable code, message, and source span, while retaining the string API during migration.

### 2. Correct and formalise source coordinates

The lexer initializes its column to one and immediately advances it when loading the first rune. This appears to report the first character of each line at column two. PHP Strom then converts the one-based parser position to a zero-based LSP position, leaving diagnostics shifted to the right.

Correct the lexer accounting and add tests for the first token, indentation, line breaks, BMP Unicode characters, and characters represented by UTF-16 surrogate pairs.

The parser should use byte offsets as its canonical internal coordinate and expose conversion at integration boundaries. PHP Strom must convert offsets to the UTF-16 code-unit positions required by LSP.

Status: the lexer now reports one-based rune columns correctly and retains byte offsets, with regression coverage for file starts, new lines, indentation, and multibyte string content. Complete AST spans and PHP Strom's byte-offset-to-UTF-16 conversion remain follow-up work.

### 3. Connect PHP Strom diagnostic settings to analysis rules

PHP Strom advertises controls for undefined symbols, undefined variables, and type errors. These values exist in its server configuration structure, but they are not updated from settings or passed to this module. Without an analysis level, `RunAnalysisRulesWithContext` runs every registered rule, including higher-level rules.

Add an explicit enabled-rule or enabled-category set to the analysis API. PHP Strom should map its user-facing settings to those categories and pass them into each analysis request. Unsupported settings should not be advertised as functional.

Status: `AnalysisContext` now accepts disabled issue codes, including codes emitted by grouped analysis passes. PHP Strom maps its undefined-symbol, undefined-variable, and type-error toggles to this selection contract. Settings for strict types, relaxed checking, mixed-type handling, and documented-type checking were removed from the advertised configuration because those semantics are not implemented.

### 4. Remove the duplicate PHP Strom parser

PHP Strom's `server/parser` implementation is not used by production language features. Its `cmd/parse-test` utility therefore measures a different parser from the one users run, which can give misleading compatibility results.

Remove the duplicate implementation or clearly isolate it as experimental. Compatibility and performance tools should call this repository through the same entry point and options used by the production language server.

### 5. Make the Go dependency reproducible

PHP Strom currently requires a zero pseudo-version and replaces it with a relative filesystem path. Its build setup clones the parser's mutable `main` branch when no sibling checkout exists and reuses that cache indefinitely.

Change this module to a canonical import path such as `github.com/ayanozturk/go-php-parser`, publish semantic version tags, and pin PHP Strom to a released version. A local `go.work` override can continue to support coordinated development without becoming part of the published dependency contract.

### 6. Add complete source spans

AST nodes currently expose only a start position. Exact editor operations such as diagnostics, selection ranges, rename, references, and highlighting need both start and end positions. Without spans, PHP Strom falls back to scanning source text around a cursor, which is fragile for Unicode and complex syntax.

Add a `Span` containing start and end byte offsets to every AST node and structured diagnostic. Keep LSP-specific coordinate conversion in PHP Strom.

## Suggested implementation order

1. Always collect parser errors and add malformed-input integration tests.
2. Correct lexer coordinates and add Unicode position contract tests.
3. Add structured parser errors and source spans.
4. Run a shared corpus through the actual PHP Strom parser adapter.
5. Wire PHP Strom diagnostic settings to analysis categories.
6. Remove its duplicate parser.
7. Publish and pin a canonical Go module version.

## Cross-repository contract tests

The long-term integration suite should cover:

- malformed PHP producing a syntax diagnostic at the failing token;
- partial code while typing without panics or hangs;
- ASCII, BMP Unicode, and surrogate-pair position conversion;
- the same parser options for compatibility metrics and production indexing;
- each PHP Strom diagnostic toggle enabling and disabling the expected rule categories;
- a pinned parser version building all supported PHP Strom target binaries.
