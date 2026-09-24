# Current implementation plan

Status: in progress

Roadmap source: short-term item 3

Scope: unify diagnostics and finish source mapping

The user explicitly reprioritized this roadmap item while short-term item 2 remains unfinished. Resume item 2 afterward unless the user changes priority again.

## Objective

Use a transport-neutral structured diagnostic with stable codes and half-open byte spans across parser, style, analysis, dependency-guard, CLI, and PHP Strom paths. Convert byte spans to each output format only at its boundary.

## Work sequence

### 1. Establish the shared diagnostic path

- [ ] Add the transport-neutral diagnostic value with filename, source, code, severity, message, and byte span.
- [ ] Route PHP Strom parser, style, and analysis diagnostics through byte spans and one LSP conversion path.
- [ ] Preserve existing parser, style, analysis, and CLI API compatibility while adding adapters.
- [ ] Identify and route dependency-guard output through the shared value.

### 2. Complete source ranges

- [ ] Add coverage for UTF-8 and UTF-16 conversion, CRLF, malformed offsets, and zero-width diagnostics.
- [ ] Migrate style rules that can identify a precise token or whitespace range from point locations to exact spans.
- [ ] Ensure every analysis diagnostic with a known node span preserves its byte offsets.

### 3. Stabilize output behavior

- [ ] Preserve deterministic ordering and stable machine-readable codes across CLI and PHP Strom.
- [ ] Add integration coverage for equivalent CLI/editor diagnostics, ranges, ordering, and malformed input.
- [ ] Document any producers that cannot yet supply a byte span and the reason.

## Acceptance gates

- Parser, style, analysis, and dependency-guard producers share one structured diagnostic representation.
- No production diagnostic relies on point-only or unstructured output where an exact span is available.
- CLI and PHP Strom publish equivalent codes, messages, severities, and source ranges.
- Range, ordering, malformed-input, and UTF-16 integration checks pass.
