# WordPress early design indicator, 2026-08-25

## Outcome

This run is accepted as an early architecture and profiling indicator, but rejected as a performance comparison. It confirms complete and deterministic file/diagnostic accounting on the selected WordPress workload and identifies allocation patterns that should be corrected during M1. It does not establish that go-php-parser is faster than Mago or that the current engine regressed against the previous snapshot.

## Reproducible source and workload

- go-php-parser commit: `b802a769b658cf6cc2290f6cde1b6bacc377951a` (clean working tree).
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Composer dependency fingerprint: `be6b7bd587d566ca1308c99b883a937efe3d11b204977499886800af9cc159c4` (`vendor/composer/installed.json`, SHA-256).
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every currently registered rule, eight workers.
- Machine: Apple M1, 8 CPUs, 8 GiB memory; macOS ARM64; Go 1.26.2.
- Cache state: fresh process per cold run, operating-system file cache uncontrolled/warm.
- Build: `go build -trimpath -ldflags='-s -w' -o benchmark ./cmd/benchmark`.
- Run: `benchmark --root test_projects/wordpress-develop --paths src,tests,vendor --excludes src/js --cold-runs 10 --warm-iterations 11 --workers 8 --json`.

Generated benchmark JSON is intentionally not checked in. The raw CPU and heap profiles were also retained only as local run artifacts; their SHA-256 values were `d437d7d8ae671f00705d025b8cf38b326a702a6bcf008fe068cf1bdd73e974ac` and `e48735a661cdc89552fc1c880376688c50b22ee3c14bc8189e26f9b5d042d11a` respectively.

## Results

All measured full-analysis iterations discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero parser failures. Every iteration emitted exactly 30,007 diagnostics.

| Phase | Mean | Median | Min | Max | Standard deviation | CV | Max peak RSS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Index only, one run | 0.643s | 0.643s | 0.643s | 0.643s | n/a | n/a | 422,068,224 bytes |
| Process-cold full analysis, ten runs | 1.215s | 1.169s | 1.065s | 1.473s | 0.133s | 10.92% | 1,038,139,392 bytes |
| Warm full analysis, ten measured iterations | 0.720s | 0.709s | 0.659s | 0.845s | 0.054s | 7.45% | 1,220,575,232 bytes |

The cold run times were 1.473s, 1.120s, 1.228s, 1.151s, 1.170s, 1.428s, 1.168s, 1.065s, 1.271s, and 1.071s. Warm measured times were 0.675s, 0.746s, 0.659s, 0.661s, 0.731s, 0.767s, 0.845s, 0.718s, 0.697s, and 0.700s.

## Context, not comparison

The earlier Go snapshot recorded 1.048s cold mean, 1.015s median, 1,083,883,520 bytes maximum peak RSS, and 29,000 diagnostics. The current result is 15.9% slower by mean and 15.2% slower by median, emits 3.5% more diagnostics, and uses 4.2% less cold peak RSS. Because the current CV is 10.92%, semantic work changed, and the runs were not interleaved, this is not sufficient evidence of a regression.

The rejected same-machine Mago 1.47.4 context recorded 3.086s mean and 1,225,932,800 bytes maximum peak RSS while emitting 218,741 findings under a materially broader strict configuration. The raw current-Go/Mago ratios are 0.394x elapsed time and 0.847x peak RSS, but they are not comparable-performance results.

## Profile findings

Three in-process full-analysis iterations allocated 3.63GB in total. The dominant application paths were:

- `ProjectIndex.MethodsDeclaredBy`: approximately 1.04GB cumulative, including approximately 0.48GB directly constructing/copying result slices and additional sorting and case-normalization allocation;
- `functionScope.clone`: approximately 0.60GB copying complete variable and property maps at branch points;
- `cloneBoolMap` and class-lineage construction: approximately 0.31GB combined;
- parsing: approximately 0.43GB cumulative, materially below the semantic-query and scope-state costs.

CPU sampling was dominated by runtime memory reclamation, clearing, and garbage-collection work. `strings.ToLower` accumulated 1.99s of CPU and repeated AST walking remained visible, but neither justified a parser-storage rewrite ahead of the identified semantic allocation work.

## Roadmap decision

The roadmap architecture remains valid, and Go remains the implementation language. Two optimizations move from later performance work into M1 prerequisites:

1. Build normalized, deterministically ordered immutable member views once per semantic snapshot and expose allocation-light internal traversal. Preserve defensive copying only at mutation-capable public boundaries.
2. Replace eager whole-map `functionScope.clone` behavior with copy-on-write, persistent, or equivalent delta-based branch state before adding broader narrowing, joins, and fixed-point analysis.

After both changes, rerun this workload interleaved with a pinned previous-engine binary. Require identical file accounting, explicitly explain diagnostic-count changes, and meet the 5% CV gate before accepting any performance claim.
