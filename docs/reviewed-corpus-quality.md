# Reviewed corpus diagnostic quality

## Release thresholds

The analyzer correctness gate is conjunctive. Every condition below must pass;
there is no weighted total score that can offset a failed condition.

1. **Differential fixtures:** zero engine mismatches and zero PHPStan reference
   mismatches in every pinned supported case. Unsupported cases must match
   their pinned PHPStan result and keep an explicit unsupported disposition.
2. **Corpus accounting:** 100% of selected PHP files are discovered and
   analyzed by both tools; zero read, parse, timeout, or analysis failures.
3. **Corpus diagnostics:** every engine-only and PHPStan-only finding must be
   reviewed and dispositioned against the selected tool versions and corpus
   revision. No unmatched diagnostic may remain unexplained at release review.
4. **Metric integrity:** publish precision, recall, F1, reviewed F1, unmatched
   counts, and provenance together. F1 is a workload-specific summary, not a
   standalone pass threshold; do not average across levels or corpora.

These thresholds make the stop condition explicit without choosing an
unsupported percentage target. An unexplained mismatch is a failed gate even
when aggregate F1 is high. A fully reviewed corpus may document intentional
behavior boundaries, but those boundaries must be reflected in a checked-in
disposition and the release notes.

## Strict differential results

On 2026-10-06, the pinned PHPStan differential packs were run with their
manifest-declared PHPStan versions. The level 5 and level 7 packs use PHPStan
2.2.16; the other packs use PHPStan 2.2.5. Across 508 cases (levels 0–8),
there were zero engine mismatches and zero PHPStan-reference mismatches. One
level 0 case is explicitly unsupported and its PHPStan finding is retained.
The remaining 507 supported cases matched exactly. The level case counts are
103 / 37 / 118 / 48 / 7 / 95 / 30 / 19 / 51.

The exact fixtures and manifest hashes are checked in under
`testdata/diagnostic-differential*/manifest.json`; `cmd/diagnostic-diff`
checks both diagnostic lists including duplicates. These fixture results pass
the first threshold, but do not by themselves establish broad corpus quality.

## PSL source corpus comparison

The pinned workload is the PSL source tree at commit
`331f3ab363508825e62c42725f96c091bcceb970` (tag 6.2.1). The measured manifest
contains the `src` directory of each of its 73 packages: 1,775 PHP files. It
excludes tests, examples, and generated/vendor code. PHPStan analyzed every
selected file without analysis errors; Tusk discovered and analyzed all 1,775.
This passes the accounting portion of threshold 2 for this workload.

The reproducible run at `2026-10-06T15:52:27Z` used PHP 8.4.24, PHPStan 2.2.16, PSL's Composer autoloader,
and the PSL PHPStan extension 2.1.0. The extension dependencies are pinned by
`testdata/phpstan-psl/composer.lock` (SHA-256
`2a29f882adbd7b973e2742bba1b2a0e1ca6d2efd8f8e25dba2649b5d5ffb7819`); the
PSL checkout's production dependencies are pinned by its upstream
`composer.lock` (SHA-256
`85e11c6f9b12cfb0794eb1ac9ab1e33c1f07d1111b9b589c7034660ef11ed111`). The
PHPStan configuration `testdata/phpstan-psl.neon` (SHA-256
`eca48699f663b6632c6b1ccdf27442d4886dfa52d814a853d118b5f237731d61`) targets
PHP 8.5, loads the two Composer autoloaders, and includes the PSL extension.
Tusk indexed the selected first-party files plus `vendor/revolt`; PHPStan
received those first-party files and the same runtime dependency through
Composer. The first-party manifest SHA-256 is
`6a955366217506f2cabe420293b4ab19265e5d38e590ba8c7ddc81aa56cc1b34`; the
combined index manifest SHA-256 is
`38c2f5b1cf88c27c33daddabfdb90cc0616cd6a0ba8742eb35bd4f2f64c014ec`. The
metric crosswalk uses the nine checked-in differential manifests listed
below. The JSON report records their individual hashes and versions. The run
command was:

```sh
go run ./cmd/phpstan-compat \
  --root test_projects/psl \
  --paths '<all 73 packages/*/src directories>' \
  --index-paths vendor/revolt \
  --levels 0,1,2,3,4,5,6,7,8 \
  --phpstan-bin ../../testdata/phpstan-psl/vendor/bin/phpstan \
  --phpstan-config ../../testdata/phpstan-psl.neon \
  --workers 3 --json --output /tmp/tusk-psl-src-all-levels.json
```

| Level | F1 | Precision | Recall | Reviewed F1 | Exact | Engine diagnostics | PHPStan diagnostics | Engine-only | PHPStan-only | Unreviewed PHPStan-only |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 12.39% | 7.37% | 38.89% | 13.46% | 7 | 95 | 18 | 88 | 11 | 9 |
| 1 | 3.92% | 2.06% | 38.89% | 4.02% | 7 | 339 | 18 | 332 | 11 | 9 |
| 2 | 3.20% | 3.17% | 3.23% | 5.83% | 15 | 473 | 464 | 458 | 449 | 422 |
| 3 | 2.54% | 2.11% | 3.21% | 3.97% | 15 | 712 | 468 | 697 | 453 | 424 |
| 4 | 2.38% | 2.11% | 2.73% | 3.97% | 15 | 712 | 550 | 697 | 535 | 506 |
| 5 | 1.95% | 1.53% | 2.67% | 2.93% | 15 | 978 | 562 | 963 | 547 | 515 |
| 6 | 4.45% | 3.38% | 6.51% | 6.30% | 39 | 1,155 | 599 | 1,116 | 560 | 515 |
| 7 | 4.12% | 3.23% | 5.69% | 5.75% | 41 | 1,269 | 721 | 1,228 | 680 | 563 |
| 8 | 4.21% | 3.26% | 5.92% | 5.81% | 43 | 1,317 | 726 | 1,274 | 683 | 563 |

The nine crosswalk inputs used in that run were:

| Level | Manifest SHA-256 | PHPStan version |
| ---: | --- | --- |
| 0 | `7d3dfe4bfc5bdce1d101a89d933d95924f417c8e882bf2ce37646f263675c8f8` | 2.2.5 |
| 1 | `654e85a0141acb2018775b7b778d378c337c31a192c2b3c99f7dcf99c48564d4` | 2.2.5 |
| 2 | `e8737c56f68ba5ea58f880e84c14ba128498bef37f5e8d05eaddfb1ebfbbc80a` | 2.2.5 |
| 3 | `188362030a5f65300deee86052688d66dd2cf42835e9389b59c3bc60343ccc11` | 2.2.5 |
| 4 | `11754f6148a84a4e2e53416644d367c42f9eb126fb5a714502feaf3f0e0cf084` | 2.2.5 |
| 5 | `08b1dffa0adf6d7339e2c5ffd775b6ef511a3b0fab63769632273b3d7d56c4cf` | 2.2.16 |
| 6 | `c8bd3eaa62d24de4bc54e912b25444841c414b90202aa6beb73558e20a8f07e7` | 2.2.5 |
| 7 | `7848c43236484c20c7ea6bad875be6dbd22673d795059b8ae55c639613a38853` | 2.2.16 |
| 8 | `981ec1f2450069c77ea8288eb3e3c01d8e0b3ef1cd1d015eaf1c10601f3c3dc1` | 2.2.5 |

**Gate status: failed.** File accounting and execution succeeded, but the
corpus still has thousands of unmatched diagnostics that have not been
individually reviewed and dispositioned. At level 8, 1,274 findings are
engine-only, 683 are PHPStan-only, and 563 PHPStan-only findings use
identifiers absent from the reviewed crosswalk. The largest engine-only
families are 423 `A.ARG.TYPE`, 231 `Level1.Core`, 147 `A.RETURN.TYPE`, 142
`Level6.MissingIterableValueType`, 91 `A.PROP.TYPE`, and 84 `Level0.Symbols`
findings. These are
concrete blockers. The numbers are a workload-specific diagnostic result,
not a claim of PHPStan parity or a score across all PHP.

The current implementation recognizes PHP magic constants and `match`'s
`default` arm, resolves function calls through imported namespace aliases,
counts concrete trait methods when checking interface contracts, and applies
PHPDoc parameter refinements alongside native parameter types in function
scopes while retaining native nullability, and accounts for top-level function
template parameters and bounds. The checked level 5, 7, and 8 differential
packs pass with zero mismatches.

The corpus checkout lacks the `bcmath` and `intl` PHP extensions required to
execute PSL. Static analysis completed without executing PSL code, and
Composer installs used `--ignore-platform-reqs`; this limitation is recorded
because those native functions are part of PSL's runtime environment.

## Reproduction

Run from the repository root with PHP 8.4 or 8.5. Fetch the pinned PSL checkout
with `go run ./cmd/fetch-test-projects`, then generate the report paths
from the pinned checkout, sorted bytewise, by selecting every `packages/*/src`
directory. Install the support lock and PSL's production dependencies first.
Save the JSON output and verify its file-manifest, index-manifest, config, and
crosswalk hashes. For fixture evidence, run each checked-in pack with the
PHPStan version pinned by its manifest. Update this page whenever a corpus,
manifest, configuration, threshold, or result changes.

```sh
mapfile -t source_dirs < <(find test_projects/psl/packages -mindepth 2 -maxdepth 2 -type d -name src | LC_ALL=C sort)
paths=$(IFS=,; printf '%s' "${source_dirs[*]#test_projects/psl/}")
(cd testdata/phpstan-psl && composer install --no-interaction --prefer-dist)
(cd test_projects/psl && composer install --no-dev --ignore-platform-reqs --no-interaction --prefer-dist)
go run ./cmd/phpstan-compat \
  --root test_projects/psl \
  --paths "$paths" \
  --index-paths vendor/revolt \
  --levels 0,1,2,3,4,5,6,7,8 \
  --phpstan-bin ../../testdata/phpstan-psl/vendor/bin/phpstan \
  --phpstan-config ../../testdata/phpstan-psl.neon \
  --workers 3 --json --output /tmp/tusk-psl-src-all-levels.json
```
