# PHPStan compatibility metric

`cmd/phpstan-compat` reports corpus-scoped diagnostic compatibility at each PHPStan level. The headline percentage is the F1 score: the harmonic mean of diagnostic precision and recall.

- An exact match requires the same normalized first-party path, start line, and a compatible PHPStan identifier.
- The engine-to-PHPStan identifier crosswalk is derived from the reviewed differential manifests.
- Clean differential cases intentionally contain no codes or identifiers and therefore add no crosswalk entry.
- Duplicate diagnostics are counted separately.
- An engine diagnostic without a reviewed identifier mapping is counted as engine-only and is also reported as unmapped.
- A run fails instead of publishing a metric when an engine file cannot be read or parsed, PHPStan returns analysis errors, a PHPStan diagnostic lacks an identifier, or PHPStan's `--debug` file accounting differs from the selected first-party manifest.

The percentage is meaningful only with its provenance: corpus revision, PHPStan version, PHP version, configuration, paths, and the recorded crosswalk source hashes. It means “diagnostic F1 on this pinned workload”, not coverage of every PHPStan rule or extension.

## Run

Run from the repository root. The paths are relative to `--root` and must select only the first-party files reported by both tools. Extra dependency sources may be supplied to the engine with `--index-paths`; PHPStan should receive the equivalent dependency context from Composer and its configuration.

```bash
go run ./cmd/phpstan-compat \
  --root /path/to/project \
  --paths src,tests \
  --index-paths vendor \
  --phpstan-bin /path/to/vendor/bin/phpstan \
  --phpstan-config /path/to/phpstan.neon \
  --json \
  --output phpstan-compatibility.json
```

The text report is intended for release notes:

```text
Level 1: X% PHPStan compatible (precision P%, recall R%, exact M)
```

Always retain precision, recall, and unmatched counts beside the headline. A rising F1 can otherwise conceal a precision/recall tradeoff.
