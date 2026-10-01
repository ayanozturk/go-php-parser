# Exact-string branch narrowing performance assessment, 2026-10-01

## Scope and result

This compares the exact-string branch-narrowing change with its parent on the
four pinned full-analysis corpora. Both binaries pass the harness's 5% cold
coefficient-of-variation gate on every corpus. Mean time changes range from
0.53% faster to 0.27% slower, within the observed run-to-run variation; this
slice shows no material full-analysis performance change. Peak RSS is similar
on all four workloads.

The candidate emits 1 additional diagnostic on PSL, 4 on Symfony, 4 on
WordPress, and no change on Magento. These are accounting deltas, not proof of
corpus-wide diagnostic parity. The focused level-5 PHPStan 2.2.5 differential
pack passes all 87 fixtures, including the newly added strict-string branch
cases.

## Reproduction inputs

- Baseline: parent commit `d8330d81`.
- Candidate: commit `0b8da0c6`.
- Go: `go1.27.1`; Linux x86_64; AMD Ryzen 7 3700X; 16 logical CPUs; 4 workers
  and `GOMAXPROCS=4`.
- Candidate binary SHA-256:
  `1b490b17d30c7fbc48d30382e8c02ee353b68caa90dcfb163b763dfea1f1f13b`.
- Baseline binary SHA-256:
  `d0e247c32307e93f3193796b4961b4f8508b07f95410a830aadc42e3202cf054`.
- Both binaries built with `go build -trimpath -ldflags='-s -w' ./cmd/benchmark`.
- Each corpus used the exact pinned revision in `test_projects/manifest.json`,
  the complete discovered path (`.`), one validation, one cold warmup, ten
  interleaved process-cold full-analysis runs per engine, 250 ms settle pauses,
  no extra runs, and no warm-loop phase. The benchmark JSON was retained under
  `/tmp/go-php-parser-perf-*.json`.
- Symfony allows exactly the known malformed fixture at
  `src/Symfony/Component/Config/Tests/Fixtures/ParseError.php`; all other read
  and parse failures remain rejected.

## Measurements

Time delta is candidate mean relative to the parent mean. The variance gate is
5%; every candidate and parent sample passed it.

| Corpus | Files discovered / parsed / failed | LOC | Bytes | Diagnostics parent → candidate | Parent mean / CV | Candidate mean / CV | Parent → candidate max RSS | Time delta |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| PSL | 3,336 / 3,336 / 0 | 316,742 | 9,436,242 | 35,798 → 35,799 | 1.744 s / 1.00% | 1.735 s / 0.85% | 175.7 → 167.4 MiB | -0.53% |
| Symfony | 10,028 / 10,027 / 1 expected | 1,847,951 | 80,347,084 | 103,922 → 103,926 | 8.170 s / 0.43% | 8.192 s / 0.55% | 1,039.1 → 1,010.0 MiB | +0.27% |
| Magento | 25,390 / 25,390 / 0 | 3,174,100 | 97,609,952 | 281,639 → 281,639 | 15.281 s / 0.45% | 15.263 s / 0.34% | 1,043.6 → 1,044.3 MiB | -0.12% |
| WordPress | 3,188 / 3,188 / 0 | 1,115,826 | 36,103,588 | 78,650 → 78,654 | 3.720 s / 0.23% | 3.725 s / 0.49% | 317.8 → 315.2 MiB | +0.15% |

Every run accounted for the same discovered, parsed, and failed file totals
within each engine and corpus. Corpus revisions are pinned in
`test_projects/manifest.json`; source corpus trees and generated benchmark JSON
remain untracked/local artifacts.
