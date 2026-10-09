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

## Named-function template-return triage (2026-10-09)

Named generic functions now substitute their declared return templates from
bare template arguments and declared callable returns. This supports arbitrary
declared names, imported function aliases, named arguments, and repeated
variadic contributions. Unresolved bindings use the declared bound or `mixed`;
structured parameter inference, dependent bounds, and unannotated callback
bodies remain unsupported. The added level-3 and level-5 clean/mismatch fixtures
check both accepted integer returns and rejected string returns/arguments.
The complete level-3 pack passes 50/50 cases against PHPStan 2.2.5 and the
level-5 pack passes 97/97 against PHPStan 2.2.16, with zero engine/reference
mismatches.

A matched engine-only before/after run at level 8 selected the same 1,775 PSL
source targets and 25 Revolt dependency files on both sides, with zero read or
parse failures. Diagnostics fell from 1,317 to 1,283: 13 `A.ARG.TYPE`,
14 `A.RETURN.TYPE`, and 7 `A.PROP.TYPE` entries disappeared, with zero additions.
This is a diagnostic delta, not a disposition of every eliminated finding:
unresolved callable templates can fall back to `mixed`, so it does not prove
complete inference or recall. Symfony (10,026 files, one existing parse-error
file) and Composer (532 files, no parse errors) had identical level-8 issue sets.
The Symfony parse error continues to block a clean full-corpus release gate.

The fresh dependency-aware PSL reference comparison completed at
`2026-10-09T21:52:52Z` using PHP 8.4.25, PHPStan 2.2.16, and the same pinned PSL,
configuration, first-party, and index manifests described above. This macOS
host uses a case-insensitive filesystem. Ordinary PSR-4 fallback autoloading
aborted with a duplicate function declaration when a missing class name mapped
to a function file. Regenerating PSL's ignored vendor autoloader with
`composer dump-autoload --no-dev --classmap-authoritative --ignore-platform-reqs`
allowed the complete reference run. Its classmap and files-map SHA-256 hashes
were `ad6ce608bab96ccb91042f826506895d112b638feff55f51158c3b715311cfa8`
and `93b810b58a1870f1a08760beba279b0553b747d248ec930a5ad50e378b50e115`.
The reference PHP process used a 1 GB memory limit. Both tools analyzed every
selected source file. This environment differs from the earlier Linux/PHP
8.4.24 comparison; reference-count changes are not attributed to the engine fix.

| Level | F1 | Precision | Recall | Reviewed F1 | Exact | Engine | PHPStan | Engine-only | PHPStan-only | Unreviewed PHPStan-only |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 8 | 4.09% | 3.20% | 5.66% | 5.68% | 41 | 1,283 | 724 | 1,242 | 683 | 563 |

The remaining engine-only argument and return families contain 410 and 133
findings respectively. The gate remains **failed** pending review/disposition.
No performance result is claimed from these correctness runs.

## Larger callable-semantics triage (2026-10-09)

Supported unannotated arrows and simple closures now infer returns using isolated
parameter scopes, implicit arrow captures, explicit closure captures, local
assignments, and bounded nesting. Known standalone callback contracts check
parameter contravariance, return covariance, and required/optional/variadic arity.
Inherited callback metadata and arbitrary declared method template names survive
binding. Initializer callbacks do not replace unrelated method-result templates.
Callable unions retain their non-callable branch, native nullable parameters keep
nullability, and supported anonymous or matching named return assertions survive
CST lowering. Integer-range assertions remain conservative pending shared type
normalization; complex control flow, references, dependent bounds, structured
binding, and the full callback/array type lattice remain partial.

The exact fixture packs pass **52/52 at level 3** (PHPStan 2.2.5) and **103/103
at level 5** (PHPStan 2.2.16), with zero engine/reference mismatches. Full tests/vet
pass on Go 1.23.12/macOS arm64; the full race suite and vet pass on Go 1.27.1.
Both Symfony and WordPress lossless identity gates pass.

A matched level-8 engine run against parent `4b0bde77` keeps all 1,775 PSL targets
and 25 Revolt index dependencies, with zero read/parse failures. Raw diagnostics
fall from **1,283 to 1,203**. Exact issue-set comparison removes 80 unique entries
(54 argument, 24 return, one property, and one invalid binary operation) and adds
none. Each side includes one duplicate raw issue. Removed findings are diagnostic
deltas, not proof that every removed finding was correctly dispositioned: some
unresolved templates conservatively become mixed.

Symfony retains 10,026 analyzed files and its one pre-existing parse error;
Composer retains 532 files with no parse errors. Exact code/location/span/message
sets change by 208 removals / 31 additions and 25 removals / 3 additions,
respectively. Of the additions, 22 Symfony and all three Composer entries replace
messages at existing code/span locations. The nine new Symfony locations comprise
two missing iterable-value annotations exposed by callback normalization, one
empty callback supplied where a session return is required, three callback
arguments involving an unindexed dependency's class hierarchy, one nullable
callback after a branch join, and two downstream method checks exposed by more
precise callback returns. The dependency, branch-join, and downstream findings
remain review work; this is **not** an unchanged-issue-set or clean-corpus claim.

The fresh dependency-aware PHPStan comparison uses the same PHP 8.4.25 / PHPStan
2.2.16 setup and autoload workaround documented above. It analyzes all selected
files and retains the same source/index/config manifests.

| Level | F1 | Precision | Recall | Reviewed F1 | Exact | Engine | PHPStan | Engine-only | PHPStan-only | Unreviewed PHPStan-only |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 8 | 4.26% | 3.41% | 5.66% | 6.01% | 41 | 1,203 | 724 | 1,162 | 683 | 563 |

Remaining engine-only argument and return findings fall from 410 / 133 to
**356 / 109**. The reviewed-corpus release gate remains **failed** pending
individual review/disposition. Matched resource evidence is recorded in
[the callable slice measurements](benchmarks/2026-10-09-callable-semantics.md).

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
