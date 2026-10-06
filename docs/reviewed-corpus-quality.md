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
2.2.16; the other packs use PHPStan 2.2.5. Across 507 cases (levels 0–8),
there were zero engine mismatches and zero PHPStan-reference mismatches. One
level 0 case is explicitly unsupported and its PHPStan finding is retained.
The remaining 506 supported cases matched exactly. The level case counts are
103 / 37 / 118 / 48 / 7 / 94 / 30 / 19 / 51.

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

The reproducible run used PHP 8.4.24, PHPStan 2.2.16, PSL's Composer autoloader,
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
| 0 | 2.01% | 1.03% | 38.89% | 2.04% | 7 | 678 | 18 | 671 | 11 | 9 |
| 1 | 1.49% | 0.76% | 38.89% | 1.50% | 7 | 922 | 18 | 915 | 11 | 9 |
| 2 | 1.97% | 1.42% | 3.23% | 2.73% | 15 | 1,055 | 464 | 1,040 | 449 | 422 |
| 3 | 1.74% | 1.19% | 3.21% | 2.30% | 15 | 1,259 | 468 | 1,244 | 453 | 424 |
| 4 | 1.66% | 1.19% | 2.73% | 2.30% | 15 | 1,259 | 550 | 1,244 | 535 | 506 |
| 5 | 1.49% | 1.03% | 2.67% | 2.00% | 15 | 1,456 | 562 | 1,441 | 547 | 515 |
| 6 | 3.49% | 2.39% | 6.51% | 4.54% | 39 | 1,633 | 599 | 1,594 | 560 | 515 |
| 7 | 3.27% | 2.29% | 5.69% | 4.21% | 41 | 1,790 | 721 | 1,749 | 680 | 563 |
| 8 | 3.35% | 2.33% | 5.92% | 4.29% | 43 | 1,842 | 726 | 1,799 | 683 | 563 |

The nine crosswalk inputs used in that run were:

| Level | Manifest SHA-256 | PHPStan version |
| ---: | --- | --- |
| 0 | `7d3dfe4bfc5bdce1d101a89d933d95924f417c8e882bf2ce37646f263675c8f8` | 2.2.5 |
| 1 | `654e85a0141acb2018775b7b778d378c337c31a192c2b3c99f7dcf99c48564d4` | 2.2.5 |
| 2 | `e8737c56f68ba5ea58f880e84c14ba128498bef37f5e8d05eaddfb1ebfbbc80a` | 2.2.5 |
| 3 | `188362030a5f65300deee86052688d66dd2cf42835e9389b59c3bc60343ccc11` | 2.2.5 |
| 4 | `11754f6148a84a4e2e53416644d367c42f9eb126fb5a714502feaf3f0e0cf084` | 2.2.5 |
| 5 | `128fe1b3c793848f3ac066cc52253f0c23c1e6693de446373d7f68c6af524ce3` | 2.2.16 |
| 6 | `c8bd3eaa62d24de4bc54e912b25444841c414b90202aa6beb73558e20a8f07e7` | 2.2.5 |
| 7 | `7848c43236484c20c7ea6bad875be6dbd22673d795059b8ae55c639613a38853` | 2.2.16 |
| 8 | `981ec1f2450069c77ea8288eb3e3c01d8e0b3ef1cd1d015eaf1c10601f3c3dc1` | 2.2.5 |

**Gate status: failed.** File accounting and execution succeeded, but the
corpus still has thousands of unmatched diagnostics that have not been
individually reviewed and dispositioned. At level 8, 1,799 findings are
engine-only, 683 are PHPStan-only, and 563 PHPStan-only findings use
identifiers absent from the reviewed crosswalk. The engine-only counts include
401 `A.ARG.TYPE`, 400 `Level0.Symbols`, 269 `Level0.ClassModel`, and 231
`Level1.Core` findings. These are concrete blockers. The numbers are a
workload-specific diagnostic result, not a claim of PHPStan parity or a score
across all PHP.

The level-1 false-positive triage fixed two recurring cases: eight PHP magic
constants and `match`'s `default` arm were being reported as undefined
constants. That removes 829 engine-only findings from levels 1–8. The focused
level-1 differential pack remains at 37 cases with zero engine and reference
mismatches.

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
