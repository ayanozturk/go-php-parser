# PHP Strom integration review

This document records the August 2026 review of `go-php-parser` as the parser and analysis engine used by the sibling `vscode-php-strom` project.

## Current relationship

PHP Strom's production indexer, semantic cache, diagnostics, and several language providers import this module through its canonical `github.com/ayanozturk/go-php-parser` path. Its former duplicate `server/parser` implementation has been removed, and `cmd/parse-test` now exercises the production parser adapter.

The review findings below are retained as historical context. The missing ESLint configuration and asynchronous diagnostic test-harness race found during that review have since been fixed. As of extension commit `5462c74` (parser `6372f1d`, pseudo-version `v0.0.0-20260831144126-6372f1de78af`), pinned and sibling-development Go tests, vet, and race suites pass, as do TypeScript lint/compile/package, all six server builds, the VS Code extension-host suite, the synthetic editor latency trace gate, structured analysis range contracts, and the current PHPStan level-boundary, DNF/nullable, callable, nested/list shape, dynamic-index, clone/coalesce/match/nullsafe, template class-string, and non-object receiver integration contracts.

## Recommended changes

### 1. Always collect parser errors

Previously, `Parser.addError` appended errors only when the parser was constructed with debug mode enabled. PHP Strom, the compatibility metrics command, and the normal concurrent scanner all construct the parser with debug mode disabled. As a result, syntax errors were silently omitted and PHP Strom's syntax-diagnostic path received an empty error list.

Parser errors must be collected independently of presentation or logging verbosity. Debug mode should control diagnostic logging only, not parser correctness.

Status: implemented in this repository. The existing `Errors() []string` API is preserved so PHP Strom remains source-compatible. A follow-up should introduce structured parser errors containing a stable code, message, and source span, while retaining the string API during migration.

### 2. Correct and formalise source coordinates

The lexer initializes its column to one and immediately advances it when loading the first rune. This appears to report the first character of each line at column two. PHP Strom then converts the one-based parser position to a zero-based LSP position, leaving diagnostics shifted to the right.

Correct the lexer accounting and add tests for the first token, indentation, line breaks, BMP Unicode characters, and characters represented by UTF-16 surrogate pairs.

The parser should use byte offsets as its canonical internal coordinate and expose conversion at integration boundaries. PHP Strom must convert offsets to the UTF-16 code-unit positions required by LSP.

Status: the lexer reports one-based rune columns correctly and retains byte offsets, with regression coverage for file starts, new lines, indentation, and multibyte string content. AST and structured analysis spans are complete. PHP Strom commit `e4c4c7b` maps those rune coordinates against the exact source into half-open UTF-16 LSP ranges, including BMP and surrogate-pair coverage. Parser errors still expose strings rather than structured spans, and mixed style-rule coordinate producers retain their legacy point ranges.

### 3. Connect PHP Strom diagnostic settings to analysis rules

PHP Strom advertises controls for undefined symbols, undefined variables, and type errors. These values exist in its server configuration structure, but they are not updated from settings or passed to this module. Without an analysis level, `RunAnalysisRulesWithContext` runs every registered rule, including higher-level rules.

Add an explicit enabled-rule or enabled-category set to the analysis API. PHP Strom should map its user-facing settings to those categories and pass them into each analysis request. Unsupported settings should not be advertised as functional.

Status: `AnalysisContext` now accepts disabled issue codes, including codes emitted by grouped analysis passes. PHP Strom maps its undefined-symbol, undefined-variable, and type-error toggles to this selection contract; extension commit `b4e2c49` includes the level-2 typed-receiver method-existence code in undefined-symbol suppression. Settings for strict types, relaxed checking, mixed-type handling, and documented-type checking were removed from the advertised configuration because those semantics are not implemented.

### 4. Remove the duplicate PHP Strom parser

PHP Strom's `server/parser` implementation is not used by production language features. Its `cmd/parse-test` utility therefore measures a different parser from the one users run, which can give misleading compatibility results.

Remove the duplicate implementation or clearly isolate it as experimental. Compatibility and performance tools should call this repository through the same entry point and options used by the production language server.

Status: PHP Strom's duplicate `server/parser` package was removed. Its `cmd/parse-test` utility now calls the production `indexer.ParseSourceForIndexWithContext` adapter and reports recovered production symbols and parser errors.

### 5. Make the Go dependency reproducible

PHP Strom currently requires a zero pseudo-version and replaces it with a relative filesystem path. Its build setup clones the parser's mutable `main` branch when no sibling checkout exists and reuses that cache indefinitely.

Change this module to a canonical import path such as `github.com/ayanozturk/go-php-parser`, publish semantic version tags, and pin PHP Strom to a released version. A local `go.work` override can continue to support coordinated development without becoming part of the published dependency contract.

Status: the module now uses `github.com/ayanozturk/go-php-parser`. PHP Strom commit `cef31c7` pins engine commit `97c5e60` through exact pseudo-version `v0.0.0-20260831124355-97c5e60e1c3d`; normal build and test targets use that module, while explicit `*-dev` targets opt into the sibling checkout through a generated Go workspace. Both paths pass tests, vet, and race checks, and the pinned parser builds all six supported server targets. Publishing a semantic version tag remains a release follow-up.

### 6. Add complete source spans

AST nodes currently expose only a start position. Exact editor operations such as diagnostics, selection ranges, rename, references, and highlighting need both start and end positions. Without spans, PHP Strom falls back to scanning source text around a cursor, which is fragile for Unicode and complex syntax.

Add a `Span` containing start and end byte offsets to every AST node and structured diagnostic. Keep LSP-specific coordinate conversion in PHP Strom.

Status: parser AST nodes and structured analysis issues carry complete spans. PHP Strom commit `e4c4c7b` consumes their end positions and converts the one-based rune coordinates against source text into half-open UTF-16 LSP ranges. Missing or invalid ends fall back safely to mapped points. Structured parser errors and style-rule range migration remain follow-ups.

### 7. Reuse immutable semantic snapshots

PHP Strom previously rebuilt an analysis context around its workspace resolver without consuming the parser's shared semantic facts, control-flow graphs, or variable-flow state. That duplicated semantic work across diagnostics and language providers and left the immutable snapshot boundary unused in the editor.

Status: snapshot consumption was implemented in PHP Strom commit `15abf7a`; incremental project-index replacement followed in parser commit `f1e06b9` and extension commit `d6680f2`; parser commit `db625ca` and extension commit `70fcf25` scope exported-change invalidation by dependency name and transitive class lineage. Parser commit `e97afef` and extension commit `d973d68` add explicit full-fallback accounting plus a bounded synthetic editor-path latency gate. Extension commit `e4c4c7b` adds structured UTF-16 analysis ranges. Diagnostics, hover, definition, and declaration consume per-document `SemanticSnapshot` instances over the latest immutable workspace project view. Matching remains conservatively lexical until generated reference facts cover every supported resolver path. Ranked remaining work is PHPStan-gated correctness (higher-level DNF, then remaining silent or extra-identifier level-0 leftovers), then `FEATURES.md` architecture rewrite, structured parser errors, and isolated-host performance measurement.

## Suggested implementation order

Historical review order; most items below are done. Current ranked work is in `docs/full-static-analyser-target.md`.

1. Always collect parser errors and add malformed-input integration tests. Done.
2. Correct lexer coordinates and add Unicode position contract tests. Done for analysis spans; parser-error strings and style-rule points remain.
3. Add structured parser errors and source spans. Spans done; structured parser errors remain.
4. Run a shared corpus through the actual PHP Strom parser adapter. Done.
5. Wire PHP Strom diagnostic settings to analysis categories. Done for the shipped toggles.
6. Remove its duplicate parser. Done.
7. Publish and pin a canonical Go module version. Canonical path and exact pseudo-version pins exist; a semantic version tag remains a release follow-up.

## Cross-repository contract tests

The long-term integration suite should cover:

- malformed PHP producing a syntax diagnostic at the failing token;
- partial code while typing without panics or hangs;
- ASCII, BMP Unicode, and surrogate-pair position conversion;
- the same parser options for compatibility metrics and production indexing;
- each PHP Strom diagnostic toggle enabling and disabling the expected rule categories;
- a pinned parser version building all supported PHP Strom target binaries.
