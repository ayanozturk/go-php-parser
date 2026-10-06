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

The recorded run used PHP 8.4.24, PHPStan 2.2.16, and
`testdata/phpstan-differential.neon` (SHA-256
`8957a9648861d3008057cbcce506a38c9ab8b8e215b8fd75675459d98ad3b68b`). The
configuration targets PHP 8.5. The report and index file-manifest SHA-256 are
both `6a955366217506f2cabe420293b4ab19265e5d38e590ba8c7ddc81aa56cc1b34`.
The metric crosswalk is built from the nine checked-in differential manifests;
their individual hashes and versions are emitted in the JSON report. The run
command was:

```sh
go run ./cmd/phpstan-compat \
  --root test_projects/psl \
  --paths '<all 73 packages/*/src directories>' \
  --levels 0,1,2,3,4,5,6,7,8 \
  --phpstan-bin /workspace/.tools/bin/phpstan \
  --phpstan-config testdata/phpstan-differential.neon \
  --workers 3 --json --output /tmp/tusk-psl-src-all-levels.json
```

| Level | F1 | Precision | Recall | Reviewed F1 | Exact | Engine diagnostics | PHPStan diagnostics | Engine-only | PHPStan-only | Unreviewed PHPStan-only |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 30.80% | 18.47% | 92.68% | 31.08% | 152 | 823 | 164 | 671 | 12 | 9 |
| 1 | 14.76% | 8.02% | 92.68% | 14.82% | 152 | 1,896 | 164 | 1,744 | 12 | 9 |
| 2 | 13.37% | 8.96% | 26.29% | 15.79% | 184 | 2,053 | 700 | 1,869 | 516 | 422 |
| 3 | 12.47% | 8.18% | 26.21% | 14.56% | 184 | 2,250 | 702 | 2,066 | 518 | 424 |
| 4 | 12.13% | 8.18% | 23.47% | 14.56% | 184 | 2,250 | 784 | 2,066 | 600 | 506 |
| 5 | 11.34% | 7.52% | 23.06% | 13.49% | 184 | 2,447 | 798 | 2,263 | 614 | 517 |
| 6 | 11.00% | 7.22% | 23.10% | 12.97% | 188 | 2,604 | 814 | 2,416 | 626 | 519 |
| 7 | 10.28% | 6.88% | 20.32% | 12.14% | 190 | 2,761 | 935 | 2,571 | 745 | 567 |
| 8 | 10.23% | 6.83% | 20.43% | 12.05% | 192 | 2,813 | 940 | 2,621 | 748 | 567 |

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
corpus has thousands of unmatched diagnostics that have not been individually
reviewed and dispositioned. The low precision and the 567 unreviewed
PHPStan-only findings at levels 7 and 8 are concrete blockers. These figures
are a baseline from this exact workload and configuration; they are not a
claim of PHPStan parity, an all-PHP score, or a historical performance
benchmark.

The run does not load PSL's own PHPStan extension/configuration. Its Composer
lock references `revolt/event-loop` 1.0.9, and this PHP environment lacks PSL's
required `bcmath` and `intl` extensions. A later exploratory level 0/2 run
loaded the Composer autoloader after installing locked production dependencies
with platform requirements ignored; it is not comparable or release evidence.
Before promoting a corpus result, pin and check in a reproducible reference
configuration and ensure Tusk and PHPStan receive equivalent dependency and
symbol context.

## Reproduction

Run from the repository root with PHP 8.4 or 8.5 and PHPStan 2.2.16. Generate
the report paths from the pinned checkout, sorted bytewise, by selecting every
`packages/*/src` directory. Save the JSON output and verify its file-manifest,
config, and crosswalk hashes. For fixture evidence, run each checked-in pack
with the PHPStan version pinned by its manifest. Update this page whenever a
corpus, manifest, configuration, threshold, or result changes.

```sh
mapfile -t source_dirs < <(find test_projects/psl/packages -mindepth 2 -maxdepth 2 -type d -name src | LC_ALL=C sort)
paths=$(IFS=,; printf '%s' "${source_dirs[*]#test_projects/psl/}")
go run ./cmd/phpstan-compat \
  --root test_projects/psl \
  --paths "$paths" \
  --levels 0,1,2,3,4,5,6,7,8 \
  --phpstan-bin "$PHPSTAN_BIN" \
  --phpstan-config testdata/phpstan-differential.neon \
  --workers 3 --json --output /tmp/tusk-psl-src-all-levels.json
```
