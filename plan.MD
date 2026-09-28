# Current implementation plan

Status: complete

Roadmap source: short-term item 3

Scope: unify diagnostics and finish source mapping

Scope is the existing parser, style, analysis, CLI, and PHP Strom diagnostic producers. No dependency-guard command or diagnostic producer exists in either checkout; if that feature is added, it must emit this shared representation.

## Objective

Use a transport-neutral structured diagnostic with stable codes and half-open byte spans across parser, style, analysis, dependency-guard, CLI, and PHP Strom paths. Convert byte spans to each output format only at its boundary.

## Work sequence

### 1. Establish the shared diagnostic path

- [x] Add the transport-neutral diagnostic value with filename, source, code, severity, message, and byte span.
- [x] Route PHP Strom parser, style, and analysis diagnostics through byte spans and one LSP conversion path.
- [x] Preserve existing parser, style, analysis, and CLI API compatibility while adding adapters.
- [x] Check for dependency-guard output; none exists in these checkouts, so there is no current producer to adapt.

### 2. Complete source ranges

- [x] Add coverage for UTF-8 and UTF-16 conversion, CRLF, malformed offsets, and zero-width diagnostics.
- [x] Migrate style rules with identifiable tokens or whitespace to exact spans; retain zero-width locations for insertion findings and legacy point-only rules without a reliable end token.
- [x] Convert analysis locations against their source so known node ranges preserve exact UTF-8 byte offsets.

### 3. Stabilize output behavior

- [x] Preserve deterministic ordering and stable machine-readable codes across CLI and PHP Strom.
- [x] Add integration coverage for shared CLI/editor diagnostic values, ranges, ordering, and malformed input.
- [x] Document producers without a current byte-span path: dependency guard does not exist; legacy rule APIs retain point locations where they cannot identify an end token.

## Acceptance gates

- Existing parser, style, analysis, CLI, and PHP Strom producer paths adapt to one structured diagnostic representation. Dependency-guard integration awaits a producer.
- Precise token and whitespace ranges are preserved; zero-width ranges remain for insertions and point-only legacy findings without a reliable end token.
- CLI and PHP Strom adapters preserve equivalent codes, messages, severities, and source ranges.
- UTF-8/UTF-16, CRLF, ordering, malformed-offset, and zero-width checks pass.
