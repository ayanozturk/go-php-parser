# PHPStan compatibility metric

`cmd/phpstan-compat` reports corpus-scoped diagnostic compatibility at each PHPStan level. The headline percentage is the F1 score: the harmonic mean of diagnostic precision and recall over exact normalized first-party path, start line, and a compatible PHPStan identifier.

- Compatible identifiers come from the reviewed differential manifests (`testdata/diagnostic-differential*/manifest.json`). Clean differential cases intentionally contain no codes or identifiers and therefore add no crosswalk entry.
- Duplicate diagnostics are counted separately.
- An engine diagnostic without a reviewed identifier mapping is counted as engine-only and is also reported as unmapped.
- PHPStan identifiers that never appear in the reviewed crosswalk at or below the measured level are **unreviewed** (framework/extension families and unimplemented core identifiers). They still affect the headline F1, but are labelled separately as `phpstanOnlyUnreviewed`. A second **reviewed F1** uses only mapped engine codes and reviewed PHPStan identifiers so framework noise is not mistaken for core-rule misses.
- A run fails instead of publishing a metric when an engine file cannot be read or parsed, PHPStan returns analysis errors, a PHPStan diagnostic lacks an identifier, or PHPStan's `--debug` file accounting differs from the selected first-party manifest.
- The reviewed fixture release gate is strict: zero engine mismatches and zero PHPStan-reference mismatches across the pinned differential manifests. Cases marked `engineSupport: unsupported` are listed separately with their exact PHPStan identifiers and must not acquire an engine diagnostic without updating that disposition.
- `cmd/phpstan-compat` emits the unmatched engine and PHPStan diagnostics with path, line, and code/identifier so corpus deltas can be reviewed and dispositioned. The command also rejects PHPStan versions that differ from the pinned manifest crosswalk version.

The percentage is meaningful only with its provenance: report-file and index-file manifest SHA-256 hashes, PHPStan version, PHP version (first line of `php -v` when available), configuration and its SHA-256, paths, and the recorded crosswalk source SHA-256 hashes. It means “diagnostic F1 on this pinned workload”, not coverage of every PHPStan rule or extension, PHPStan parity, or a published percentage. The strict fixture gate is the release threshold; corpus F1 remains a separate workload metric and should not be reported for a zero-diagnostic corpus without its zero denominators.

## Run

Run from the repository root. `--paths` selects the first-party files in the compatibility denominator; both tools must report diagnostics only from that manifest. `--index-paths` supplies extra PHP sources indexed for engine symbol resolution only (for example `vendor`). Those files are not type-checked and do not enter the denominator. The engine scopes analysis and reported diagnostics to the reportable first-party files.

The crosswalk glob is resolved from the current working directory, then from the Go module root if the cwd has no matches.

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
Level 1: X% PHPStan compatible (precision P%, recall R%, exact M, engine-only E, PHPStan-only S, unmapped U, PHPStan-unreviewed Rv; reviewed F1 Y%)
```

Always retain precision, recall, unmatched counts, unreviewed count, and reviewed F1 beside the headline. A rising F1 can otherwise conceal a precision/recall tradeoff or hide framework noise in the PHPStan-only bucket.
