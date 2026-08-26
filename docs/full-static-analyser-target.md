# Full Static Analyser and Mago-Class Performance Target

## Status

- Project target: approved
- Target repositories: `go-php-parser` and `vscode-php-strom`
- Baseline date: 2026-08-23 (Europe/London)
- Benchmark references: Mago for performance and modern PHP type analysis, PHPStan and Psalm for diagnostic depth, and PHPCS for source-style breadth

## Mission

Evolve `go-php-parser` from a parser with independent style and analysis rules into a complete, production-grade PHP static analyser, and deliver it through PHP Strom as a fast incremental editor experience.

The finished analyser must be competitive with Mago on performance without obtaining speed by doing less work. It must combine:

- broad PHP 8.x syntax support;
- project-wide symbol and dependency resolution;
- flow-sensitive and interprocedural type analysis;
- PHPDoc, generic, conditional, callable, and shape types;
- useful framework-aware extension points;
- predictable diagnostics with low false-positive and false-negative rates;
- safe operation on malformed and partially typed code;
- cold full-project analysis performance comparable to Mago;
- low-latency incremental analysis for PHP Strom.

This is a correctness, coverage, reliability, and performance target. None of those dimensions may be traded away invisibly to improve another.

## Definition of success

The project may claim that it is a full static analyser only when all of the following are true:

1. It parses supported PHP versions and builds a complete semantic model for the configured project and dependencies.
2. It performs control-flow-sensitive type checking across functions, methods, closures, traits, inheritance, and common dynamic PHP patterns.
3. It supports the PHPDoc type constructs used by the reference corpus, including generics and type aliases, with documented diagnostics for unsupported constructs.
4. It can run as a standalone cold analyser and as an incremental language-server engine.
5. It completes the reference corpora without a panic, timeout, data race, or silent file omission.
6. Its diagnostic quality passes the correctness gates in this document.
7. Its full-analysis performance passes the comparable-performance gate in this document.

Passing a fast symbol-index benchmark does not satisfy this definition.

## Benchmark reference and current evidence

Mago's public site reports a 1.46-second static-analysis run over approximately 7 million LOC. The benchmark is a cold analyzer run against WordPress and is configured to scan `src`, `tests`, and `vendor`. Its strict configuration enables dead-code, unused-definition, throws, missing-type-hint, finality, and related checks.

Primary sources:

- [Mago homepage and headline benchmark](https://mago.carthage.software/latest/en/)
- [Mago benchmark methodology](https://mago.carthage.software/1.47.2/en/benchmarks/)
- [Reproducible PHP toolchain benchmark suite](https://github.com/carthage-software/php-toolchain-benchmarks)
- [WordPress Mago benchmark configuration](https://raw.githubusercontent.com/carthage-software/php-toolchain-benchmarks/main/project-configurations/wordpress/mago.toml)
- [Published benchmark data](https://carthage-software.github.io/php-toolchain-benchmarks/latest.json)

The published result was last refreshed on 2026-04-15. The 1.46-second entry is for Mago 1.20.1, while GitHub's live release metadata identified Mago 1.47.4 as current on 2026-08-25. The public methodology does not identify the benchmark CPU and memory configuration. Absolute times must therefore be reproduced on the same machine before making a comparative claim; the rejected 1.47.4 rebaseline recorded in action 7 does not replace the headline with a valid relative result.

### Local machine used for the initial baseline

- Apple M1, 8 cores (4 performance and 4 efficiency)
- 8 GB memory
- Go 1.26.2, `darwin/arm64`

### Existing symbol-index baseline

PHP Strom's existing `cmd/benchmark-indexer` is not a static-analysis benchmark. It excludes `vendor`, skips function bodies, and measures file discovery, parsing for symbols, and index construction with at most four workers.

On a local 4,042,704-LOC corpus, five cold-process invocations produced:

- times: 4.050s, 4.222s, 5.458s, 2.909s, and 3.363s;
- median: 4.050s;
- median throughput: approximately 998,000 LOC/s;
- observed throughput range: approximately 741,000 to 1,390,000 LOC/s;
- observed Go system memory: approximately 751 to 825 MB.

Mago's headline result is approximately 4.79 million LOC/s. It is therefore about 4.8 times the throughput of this easier symbol-only local benchmark, before accounting for hardware differences.

### Initial full-diagnostics baseline

An ephemeral harness reproduced PHP Strom's cold sequence: symbol indexing followed by the configured two-worker workspace diagnostics pass.

On a local 249,780-LOC corpus:

- symbol indexing: 0.253s;
- diagnostics: 0.903s;
- total: 1.156s;
- total throughput: approximately 216,000 LOC/s;
- diagnostics produced: 34,091;
- failures: zero;
- end-of-run Go memory: 184.6 MB heap and 212.9 MB system memory.

The diagnostic count is not a quality result. It requires classification against expected findings before it can support a correctness claim.

On the 4,042,704-LOC corpus:

- symbol indexing completed in 4.637s;
- full diagnostics did not complete;
- concurrent analysis reached a nil-pointer panic through per-file project-index construction.

This failure is a release-blocking scalability defect for the full-analyser target. No large-corpus full-analysis throughput is currently claimed.

**Update:** a stress test with a synthetic 400+ file corpus of interdependent classes, run under `go test -race` with concurrent project-index construction and analysis, reproduced a crash in this code path: `finalMethodInAncestors`, `finalConstantInAncestors`, `consistentConstructorInAncestors`, `collectAbstractMethods`, `collectUnimplementedParentAbstractMethods`, and `isSubclassOf` in `analyse/phpstan_level0_class_model.go` and `analyse/phpstan_level0_helpers.go` recursed over a class's `Extends`/`Implements` chain with no cycle detection. A self-referential or mutually cyclic `extends`/`implements` chain — which parses as valid PHP syntax even though PHP itself would reject it at runtime — caused unbounded recursion and a stack-overflow crash. All six helpers now carry a `seen`-set guard against revisiting a class, matching the cycle-safe pattern already used elsewhere in the package (e.g. `ProjectIndex` ancestor walks, `isThrowableClass`). A regression test, `TestLevel0CyclicClassHierarchyDoesNotHang` in `analyse/phpstan_level0_rule_test.go`, asserts analysis completes within a bounded timeout on self-referential and mutually cyclic fixtures. This closes the specific stack-overflow crash class reproduced here; it has not been confirmed to be the exact same failure signature (a nil-pointer panic) originally observed on the 4,042,704-LOC corpus, so the large-corpus run should be repeated to confirm the release-blocking defect is fully resolved before removing this note.

### Early WordPress design indicator, 2026-08-25

A provisional benchmark on clean commit `b802a769b658cf6cc2290f6cde1b6bacc377951a` exercised the current M1 engine against the pinned WordPress corpus (`src`, `tests`, and `vendor`, excluding `src/js`) with eight workers. All ten process-cold runs and all ten measured warm iterations accounted for 5,357/5,357 files, 1,451,208 LOC, 47,344,277 bytes, zero parse failures, and exactly 30,007 diagnostics.

| Phase | Mean | Median | Min | Max | CV | Max peak RSS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Index only, one run | 0.643s | 0.643s | 0.643s | 0.643s | n/a | 422,068,224 bytes |
| Process-cold full analysis, ten runs | 1.215s | 1.169s | 1.065s | 1.473s | 10.92% | 1,038,139,392 bytes |
| Warm full analysis, ten measured iterations | 0.720s | 0.709s | 0.659s | 0.845s | 7.45% | 1,220,575,232 bytes |

The cold mean is 15.9% and the median 15.2% above the earlier 2026-08-25 Go snapshot (1.048s mean, 1.015s median), while diagnostic output increased from 29,000 to 30,007 and cold peak RSS decreased by 4.2%. The 10.92% cold CV fails the 5% contract, the engine's semantic coverage remains materially below Mago's strict configuration, and the tools were not interleaved in this indicator run. It therefore establishes stability and profiling direction only; it is not an accepted regression measurement or a Mago performance comparison. The raw ratio to the rejected same-machine Mago context is 0.394x elapsed time and 0.847x peak RSS, but neither ratio supports a parity claim.

A three-iteration profile allocated 3.63GB in total. The `ProjectIndex.MethodsDeclaredBy` path accounted for approximately 1.04GB cumulatively, including repeated result construction, sorting, and case normalization; `functionScope.clone` allocated approximately 0.60GB by copying branch-state maps. Runtime memory clearing, reclamation, and garbage-collection work dominated the CPU samples, while parsing accounted for a much smaller share of allocations. This confirms two early M1 design changes:

- internal semantic queries must use immutable, pre-normalized, deterministically ordered views or iterators without cloning and sorting on every call; defensive copying remains required only at mutation-capable public boundaries;
- branch-sensitive flow state must move from eager whole-map cloning to a measured copy-on-write, persistent-map, or equivalent delta representation before broader narrowing and fixed-point analysis multiply the cost.

The result does not justify changing implementation language or redesigning AST storage yet. Shared semantic facts, normalized symbol identities, allocation-light resolver traversal, and persistent flow state remain the priority. The durable result summary and reproduction details are recorded in `docs/benchmarks/2026-08-25-wordpress-early-design-indicator.md`; generated JSON remains a local or CI artifact and is not committed.

### Post-allocation interleaved rebaseline, 2026-08-25

Commit `6ac1a40` implemented immutable precomputed method views, allocation-light internal traversal, copy-on-write function-scope maps, and a hardened benchmark protocol. Two WordPress attempts interleaved it with the previous engine at `b802a76`. Every validation and measured run retained identical accounting: 5,357/5,357 files, 1,451,208 LOC, 47,344,277 bytes, zero parse failures, and 30,007 diagnostics.

Across 10-run and 20-run attempts the candidate's cold mean and median were approximately 23% lower and maximum peak RSS was approximately 5% lower. Both attempts remain rejected: candidate CV was 6.08% and 6.94%, while baseline CV was 5.21% and 5.18%. This is repeated directional evidence that the allocation design is beneficial, but it is not an accepted performance baseline. The benchmark protocol correctly rejected favorable results that failed its stability contract. The durable result summary is recorded in `docs/benchmarks/2026-08-25-wordpress-post-allocation-rebaseline.md`; generated JSON remains a local or CI artifact and is not committed.

## Comparable-performance contract

All performance claims must use a checked-in, reproducible harness and record:

- repository URL and exact commit or tag;
- dependency lock state and setup command;
- included and excluded paths;
- PHP version and analyser configuration;
- analyser commit, Go version, build flags, and environment variables;
- CPU, logical and physical core count, memory, operating system, and architecture;
- cold or warm cache state;
- run count, mean, median, minimum, maximum, standard deviation, and coefficient of variation;
- peak RSS for the whole process tree;
- files discovered, files analysed, LOC, bytes, and diagnostics emitted;
- parse failures, analysis failures, panics, timeouts, and skipped files;
- rule and feature coverage enabled for each compared tool.

### Required workloads

The shared external suite must include the same projects used by the Mago benchmark:

- `php-standard-library/php-standard-library`;
- `WordPress/wordpress-develop`;
- `magento/magento2`.

The project must also retain representative framework corpora for Symfony, Laravel, Doctrine, Composer, Drupal, PHPUnit, and modern PHP syntax. Private corpora may be used to find defects but must not be required to reproduce a public performance claim.

### Cold-run protocol

1. Build a release binary once, outside the timed region.
2. Remove analyser caches before every measured run.
3. Preserve operating-system cache state consistently across compared tools, or explicitly measure and label both filesystem-cold and process-cold variants.
4. Run tools interleaved to reduce drift from temperature and background load.
5. Use at least ten measured runs after an unmeasured validation run.
6. Reject a comparison when CPU stability, file counts, configuration, or semantic coverage differs materially.

### No-benchmark-gaming rules

- Do not exclude dependencies that the compared configuration includes.
- Do not skip function bodies, tests, generated PHP, or error-producing files unless every compared tool receives the same exclusion.
- Do not disable expensive rules only for the measured analyser.
- Do not count indexing as full analysis.
- Do not omit startup, configuration loading, dependency discovery, parsing, result reduction, or diagnostic construction from a cold full-analysis time.
- Report crashes, skipped files, and timeouts as failed runs, not as faster results.
- Keep correctness and rule-coverage results beside every performance result.

## Final performance target

On the same machine, corpus revision, paths, dependencies, cache state, and mutually supported strict-analysis configuration:

- cold full-analysis mean must be no more than 1.5 times Mago's mean;
- stretch target: equal or faster than Mago;
- peak RSS must be no more than 1.25 times Mago's peak RSS unless a documented interactive-mode trade-off is approved;
- coefficient of variation should be at most 5%;
- 100% of selected PHP files must be accounted for;
- there must be zero panics, fatal analysis failures, data races, or per-file timeouts;
- diagnostic correctness and coverage gates must pass in the same build.

Against the currently published 1.46-second result, the provisional cold target is at most 2.19 seconds and the stretch target is at most 1.46 seconds. These absolute values are illustrative; the authoritative comparison is the contemporaneous same-machine relative result.

## Diagnostic correctness and coverage gates

Performance is not comparable unless the analyser performs comparable semantic work.

### Parser and source model

- Support all valid syntax for the declared PHP 8.x range.
- Distinguish pure PHP, mixed templates, generated files, and unsupported extensions in compatibility reports.
- Add complete byte spans to AST and diagnostics, with tested UTF-16 conversion at the LSP boundary.
- Preserve useful partial trees for incomplete editor input without panics or uncontrolled cascades.
- Reach at least 99.99% parse compatibility on the checked-in representative corpus before a release candidate; every remaining failure must be classified.

### Semantic model

- Namespaces, imports, aliases, constants, functions, classes, interfaces, traits, enums, properties, methods, and promoted properties.
- Inheritance, trait composition and adaptations, interface contracts, visibility, finality, readonly semantics, and variance.
- Composer autoload roots, project source roots, configured dependency roots, stubs, and PHP-version-specific built-ins.
- Stable symbol identities, declaration spans, references, and an immutable project snapshot suitable for parallel queries.

### Type system

- Native union, intersection, nullable, literal, `never`, `void`, `mixed`, `object`, iterable, callable, and class-string types.
- Array/list shapes, keyed arrays, non-empty variants, integer ranges, literal strings, and object shapes where supported by reference annotations.
- PHPDoc templates, bounds, covariance and contravariance, generic inheritance, type aliases, imports, conditional types, indexed access, key/value projections, and callable signatures.
- `self`, `static`, parent, late-static binding, `$this`, and fluent-return semantics.
- Sound normalization, subtyping, narrowing, widening, substitution, and recursion limits.

### Flow and interprocedural analysis

- Control-flow graphs for functions, methods, and closures.
- Reachability, definite assignment, return completeness, throw paths, and termination.
- Branch narrowing from comparisons, assertions, `instanceof`, null checks, array-key checks, match/switch, coalescing, and short-circuit expressions.
- Loop fixed points with bounded convergence.
- Call argument and return checking, named and unpacked arguments, by-reference effects, variadics, generators, closures, and first-class callables.
- Property initialization and mutation, constructor flow, readonly constraints, and dynamic-property rules.
- Interprocedural summaries with dependency invalidation rather than whole-project recomputation for every edit.

### Framework and dynamic-code support

- Extension contracts for dynamic return types, assertions, method/property reflection, generated members, service containers, and ORM repositories.
- First-party compatibility packs should prioritize Composer, PHPUnit, Symfony, Doctrine, and Laravel patterns used by the reference corpora.
- Dynamic support must be testable and must not silently weaken ordinary type checking.

### Quality measurement

- Differential fixtures against PHPStan, Psalm, and Mago for mutually supported semantics.
- Curated true-positive, false-positive, and false-negative suites.
- Every fixed user-reported false positive receives neutral synthetic regression coverage.
- Diagnostic stability checks ensure unrelated edits do not reorder or rewrite findings nondeterministically.
- Rule documentation states intent, severity, configuration, known limitations, and reference behavior.

## Target architecture

### Single-source pipeline

Each file should be read and lexed once per content version. Parsing, symbol extraction, semantic facts, diagnostics, and editor features must share the resulting immutable snapshot.

The current cold workspace path first parses with function bodies skipped and later reads and parses files again for diagnostics. Remove that duplication for full-analysis runs. Retain a deliberately lighter index-only mode only when it is named and measured separately.

### Compact representations

Go can reach the target, but a pointer-heavy heap graph will make garbage collection and cache locality limiting factors. The preferred design is:

- token and node storage in contiguous slices;
- integer node, type, symbol, and string IDs;
- interned identifiers and normalized types;
- compact tagged unions rather than pervasive interface values where profiling supports the change;
- file- or worker-owned allocation slabs for short-lived analysis facts;
- explicit lifetime boundaries so whole phases can release memory together;
- bounded pools only where benchmarks prove reuse is beneficial.

### Shared semantic facts

Independent rules currently traverse the same AST repeatedly. Build reusable per-file facts in coordinated passes:

- scope and declaration tables;
- control-flow graph and reachability;
- expression types and narrowing facts;
- reads, writes, calls, throws, and return effects;
- suppression and source-location maps.

Rules should query these facts instead of rebuilding them. A fused visitor is preferred where it reduces traversal and allocation without coupling unrelated diagnostics.

### Project graph and concurrency

- Build one immutable project graph per revision.
- Never rebuild the entire project index as an uncoordinated per-file cache fallback.
- Fix the large-corpus project-index panic before increasing concurrency.
- Separate CLI and LSP scheduling policies: batch analysis may use all effective cores, while interactive mode reserves capacity for editor requests.
- Partition work by strongly connected dependency components when inter-file ordering matters.
- Use deterministic reduction so concurrency does not change diagnostic output.

### Incremental analysis

- Hash file content and semantic exports separately.
- Reanalyse dependants only when exported semantic facts change.
- Cache parsed snapshots, declarations, type summaries, and diagnostics by content and configuration fingerprint.
- Bound cache memory and expose cache hit, invalidation, and eviction metrics.
- Cancel obsolete editor work promptly and never publish results for stale document versions.

## Delivery milestones

### M0 — Reproducible baseline and stability

Exit criteria:

- Add a standalone production `analyze` command that exercises the same engine as PHP Strom.
- Add the external three-project benchmark harness with pinned revisions and generated machine-readable reports.
- Measure index-only, cold full analysis, warm full analysis, and incremental edits separately.
- Reproduce and fix the large-corpus project-index panic with adversarial regression coverage.
- Account for every discovered file and classify parser failures.
- Record rule coverage and diagnostic counts beside timing and RSS.

Progress: the production `analyze` command now parses each selected file once, constructs one immutable `SemanticSnapshot`, runs the same registered analysis engine used by PHP Strom, emits deterministic diagnostics, accounts for parser/read failures, and returns stable clean/finding/infrastructure exit codes. Literal end-to-end pipeline equivalence with PHP Strom remains an extension-side M1 integration because the extension currently supplies its own workspace overlay, stubs, PHP version, and disabled-code context.

### M1 — Complete semantic foundation

Exit criteria:

- Complete source spans and structured diagnostics.
- Immutable project graph and stable symbol identities.
- Allocation-light internal resolver traversal with normalized keys and stable member ordering constructed once per immutable snapshot; ordinary analysis queries must not defensively clone or sort whole member collections.
- Control-flow graph, reusable semantic-fact store, and foundational type operations.
- Copy-on-write, persistent, or equivalently allocation-bounded branch state, with profiling evidence that flow-sensitive branching no longer eagerly copies complete variable and property maps.
- Full PHPStan level 0 behavior for the agreed corpus and documented progress through levels 1–3.
- Cold WordPress analysis completes reliably in at most 60 seconds and 2 GB peak RSS on the reference machine.

### M2 — Broad type-analysis capability

Exit criteria:

- Generic PHPDoc, shapes, callable types, flow narrowing, property initialization, trait/interface contracts, and interprocedural summaries meet their fixture gates.
- PHPStan levels 0–6 are substantially covered, with explicit coverage metrics rather than name-only claims.
- Symfony, Doctrine, Composer, and PHPUnit compatibility packs pass their integration suites.
- Cold WordPress analysis completes in at most 10 seconds and 1.5 GB peak RSS on the reference machine.

### M3 — Full-analyser release candidate

Exit criteria:

- The parser and semantic coverage gates in this document pass.
- Differential false-positive and false-negative suites meet release thresholds chosen from reviewed evidence.
- Magento and PSL complete under the same reliability contract as WordPress.
- Cold WordPress analysis completes in at most 5 seconds and 1.25 GB peak RSS on the reference machine.
- PHP Strom incremental analysis meets the latency budget below.

### M4 — Comparable-performance release

Exit criteria:

- Cold full analysis is at most 1.5 times the contemporaneous Mago mean on all required workloads where both tools complete.
- The WordPress stretch investigation records the remaining gap to equal-or-faster performance.
- Peak RSS, variance, file accounting, reliability, and semantic-coverage gates pass.
- Results are reproduced in CI or on a documented stable benchmark host and published with raw machine-readable evidence.

## PHP Strom interactive targets

Batch speed alone is insufficient for an editor analyser. On a representative warm workspace and supported development machine:

- open-document diagnostic latency: p50 at most 50ms, p95 at most 100ms;
- incremental edit affecting only local facts: p95 at most 100ms;
- incremental edit changing exported symbols: p95 at most 300ms for the affected dependency slice;
- cancellation acknowledgement: p95 at most 25ms;
- no stale diagnostics after a newer document version is accepted;
- background work must not prevent completion, hover, definition, or signature-help responses from meeting their own latency budgets.

These targets require a dedicated trace-based benchmark; synthetic single-file timing alone is insufficient.

## Verification and release gates

Every milestone delivery must run the checks appropriate to its changes, including:

- engine unit, integration, race, fuzz-seed, and compatibility tests;
- extension pinned-module and sibling-development tests;
- deterministic benchmark smoke tests in normal CI;
- scheduled full benchmark and longer fuzz runs;
- `go vet`, diff checks, TypeScript lint/compile/package, and extension-host tests;
- cross-platform builds for every shipped language-server target;
- performance comparison against the previous release using interleaved runs;
- investigation of regressions larger than 5%, with no unsupported improvement claim inside normal run noise.

A release must not advance the parser version pinned by PHP Strom until the engine commit is pushed and the pinned and sibling-development paths have both passed.

## Repository responsibilities

### `go-php-parser`

- parser, AST/spans, semantic IR, type system, project graph, analysis engine, diagnostics, standalone CLI, benchmark protocol, and correctness suites;
- stable APIs for immutable snapshots, cancellation, configuration, and machine-readable reports;
- public benchmark evidence and analyzer capability documentation.

### `vscode-php-strom`

- workspace discovery, dependency/configuration mapping, versioned document snapshots, incremental scheduling, cancellation, LSP conversion, diagnostic publication, and editor latency benchmarks;
- pinned production dependency plus explicit sibling-development workflow;
- integration tests proving the editor and standalone CLI use equivalent analysis semantics.

## Immediate next actions

1. ~~Add the checked-in cold/full/incremental benchmark command and result schema.~~ Done: `cmd/benchmark` (`go run ./cmd/benchmark --root <dir> --json`) measures index-only, process-cold full analysis (subprocess-per-run), and warm-loop full analysis, with mean/median/min/max/stddev/CV, file accounting, diagnostic counts, and RSS (OS rusage + in-process `runtime.MemStats.Sys`). Incremental timing is explicitly reported unsupported pending an incremental-invalidation API (M1/M2). Pinning to the exact Mago benchmark projects/config (action 2) and wiring it into CI as a scheduled job (`.github/workflows/benchmark.yml`) are both now done too.
2. ~~Pin and automate the Mago benchmark projects and configuration.~~ Done, with an unplanned but higher-priority fix along the way: `test_projects/{composer-src,drupal,laravel,phpunit,symfony}` were committed as bare git "gitlinks" (mode `160000`, like submodule entries) with no accompanying `.gitmodules`, so a fresh clone of this repository silently produced five **empty** directories — every benchmark/compat-metrics claim referencing them (including the determinism and profiling fixes recorded below) was only reproducible on checkouts that happened to already have those nested `.git` directories on disk. Fixed by untracking the gitlinks (`git rm --cached`), ignoring `test_projects/*` except a new `test_projects/manifest.json`, and adding `cmd/fetch-test-projects` (`go run ./cmd/fetch-test-projects`), which fetches each project's exact pinned commit (a single-commit shallow fetch, not a full clone) into `test_projects/<name>` and is idempotent (skips projects already at the pinned commit). The manifest now records exact commits for the existing five corpora plus the three workloads the Mago benchmark itself requires: `php-standard-library/php-standard-library` (pinned to tag `6.2.1`), `WordPress/wordpress-develop` (tag `7.1.0`), and `magento/magento2` (tag `2.4.9`) — the latest stable release of each as of 2026-08-24. Note Mago's own benchmark suite (`carthage-software/php-toolchain-benchmarks`) does not itself pin exact commits for these three projects or publish them in its results feed; it only pins `php-standard-library` via a composer semver range and otherwise tracks each project's default branch, pinning only tool versions. This project's manifest pins exact commits for all eight corpora so every run here is independently reproducible regardless of upstream drift. `wordpress-develop` and `magento2` have now been fetched and benchmarked (see the note below on the two parser gaps this surfaced). Done: `.github/workflows/benchmark.yml` runs weekly (plus `workflow_dispatch`) and calls `cmd/fetch-test-projects`, then `cmd/compat-metrics` (whole `test_projects` root) and `cmd/benchmark` (for the three required Mago workloads: `psl`, `wordpress-develop`, `magento2`), uploading all JSON reports as build artifacts. Kept off the per-PR path since the corpora are large and the benchmark's process-cold phase re-execs a subprocess 10 times per corpus by default.
3. ~~Minimize and fix the large-corpus project-index panic before further performance tuning.~~ Fixed the reproduced stack-overflow variant of this defect: six ancestor-walking helpers recursed over `extends`/`implements` chains with no cycle detection, so a self-referential or mutually cyclic class hierarchy caused unbounded recursion. All six now carry `seen`-set guards, with a regression test (`TestLevel0CyclicClassHierarchyDoesNotHang`). `cmd/benchmark` exercised this exact code path against the full `test_projects/symfony` corpus (10,478 files, 1.8M LOC) with no crash. The original report described a nil-pointer panic specifically; that signature has not been independently reproduced, so treat this as fixing the reproduced crash class rather than a confirmed root-cause match until re-verified on the original 4M-LOC corpus.
4. ~~Profile allocations, GC, parsing duplication, project-index construction, and per-rule AST walks.~~ In progress: `cmd/benchmark` now supports `--cpuprofile`/`--memprofile`/`--profile-iterations`, running the real parse+index+analyse pipeline in-process (no subprocess boundary) so `go tool pprof` sees genuine allocation and CPU data. A pass against `test_projects/symfony` (10,478 files) found: (a) `PHPStanLevel0Rule.checkLanguage`/`checkSymbolsAndCalls` dominate CPU and allocations (~19% of total CPU, ~1.5GB combined allocated over 5 iterations) as the single largest rule by far; (b) `functionScope.clone` (`return_type_rule.go`) allocates ~580MB by rebuilding two full maps (`variables`, `properties`) per scope split for flow-sensitive narrowing — a candidate for copy-on-write or persistent-map representation; (c) **fixed** a lines-cache thrashing bug: `sharedcache.SplitLinesCached`'s eviction threshold (previously 10,000) tracked cumulative stores rather than live entries and was smaller than several of this project's own benchmark corpora (e.g. `test_projects/drupal` at 10,856 files), so the cache cleared itself mid-run and forced repeated re-splitting of the same files (565MB / 10.5% of total allocations in the profiled run). The counter now tracks live entries (decremented on individual eviction) and the threshold was raised to 200,000 as a memory safety valve rather than a per-corpus budget; total allocations on the same symfony profiling run dropped from 5,361MB to 4,938MB (-8%) with `SplitLinesCached` no longer appearing in the top allocators. Remaining items ((a) and (b) above) are structural rule/type-analysis work, not quick fixes, and are left as follow-ups.
5. **Done — design the immutable semantic snapshot and shared fact-store interfaces.** `analyse/semantic_snapshot.go` owns the mutable `ProjectIndex` behind a read-only `SymbolResolver`, returns defensive copies of slice-backed symbol metadata, and exposes a deterministic byte-span-keyed `SemanticFactReader`. Fact construction rejects invalid spans and duplicate keys rather than allowing order-dependent overwrites. `ProjectIndex` assigns stable case-insensitive semantic IDs and source declaration spans while indexing classes, functions, methods, properties, class constants, and enum cases; inherited member resolution retains the declaring symbol's identity and location, while built-ins have stable IDs with explicitly empty source locations. Completing this exposed and fixed missing `EndPos` values for nested class/interface declarations, the symbol-only `SkipFunctionBodies` path, and intermediate postfix receivers. `A.RETURN.TYPE`, `A.ARG.TYPE`, and `A.PROP.TYPE` consume exact file/span `inferred-type` facts and retain existing inference as a safe fallback for absent, empty, or nonmatching facts; `A.ARG.COUNT` shares fact-aware method-receiver resolution. Snapshot construction produces return, argument, assignment, receiver, condition, literal, variable, property, call, and construction facts in one branch- and assignment-aware function-body traversal, attaches the enclosing function or method's stable symbol ID, and preserves an explicitly supplied fact at the same source span. Filename-aware hover APIs consume those facts while the original APIs remain source-compatible and retain inference fallback when no filename is supplied. The PHPStan class-model, symbol, and invocation passes use resolver queries for duplicates, members, constants, constructor lineage, subclass checks, trait composition, ancestor-final checks, consistent constructors, visibility, and required-method collection; none reads `ProjectIndex` or its mutable maps directly. `AnalysisContext` now carries only the read-only resolver and fact-reader contracts, registered-rule context cloning preserves both, and a missing resolver is bootstrapped locally without retaining a mutable project handle. Adversarial tests cover input/result mutation isolation, stable ordering and IDs, inherited ownership, declaration and receiver spans, skipped bodies, built-ins, concurrent reads, scope-aware generated facts, externally supplied fact reuse across migrated diagnostics and hover queries, resolver-only class-model/symbol/invocation diagnostics, registered-rule fact propagation, legacy API behavior, precedence, cycle safety, and safe fallback. The semantic boundary is now ready for the control-flow graph and broader foundational type operations required by M1.
6. ~~Establish the first diagnostic differential suite and publish a capability matrix.~~ Done: `cmd/diagnostic-diff` validates exact sorted diagnostic identifiers against the checked-in manifest in `testdata/diagnostic-differential`, with five initial level-0 cases covering unknown classes, unknown functions, function argument counts, always-undefined variables, and a clean known-symbol control. Engine-only mode is part of the Go test gate; a full run invokes an explicitly supplied PHPStan binary, validates its identifiers, and records its reported version without silently downloading an external tool. `docs/analyser-capability-matrix.md` distinguishes differential-gated partial capabilities, unit-tested areas not yet in the differential pack, unsupported milestone areas, and the dependency blocking each gap.
7. ~~Rebaseline on the same machine against the current Mago release before setting milestone dates.~~ Done, but rejected as a milestone comparison: the 2026-08-25 run used verified Mago 1.47.4 and pinned corpora on the same Apple M1 host. Validation rejected PSL (65/3,361 parser failures) and Magento (15/41,848). WordPress completed 5,357/5,357 files across ten interleaved process-cold runs, but go-php-parser CV was 6.90%, Mago CV was 5.11%, Mago's count formatter did not expose file accounting, and semantic output differed materially (29,000 vs 218,741 diagnostics). No speed or milestone-date claim is supported. The run also exposed and fixed a benchmark defect: cold timing previously began after discovery/read/parse; it now measures at the parent process boundary and includes the entire cold pipeline. Raw results and the rejection rationale are recorded under `docs/benchmarks/2026-08-25-mago-1.47.4-rebaseline.{md,json}`.
8. **Done — reduce semantic-query and branch-state allocation before expanding M1 analysis breadth.** Commit `6ac1a40` replaced repeated internal `MethodsDeclaredBy` copying, sorting, and parameter cloning with normalized immutable member views and allocation-light traversal while preserving defensive copies at the public boundary. `functionScope.clone` now shares variable and property maps until first write, with isolated copy-on-write behavior for original, child, sibling, and chained scopes. Focused allocation, immutability, and race tests cover both changes. Two interleaved WordPress attempts preserved exact accounting and repeatedly showed approximately 23% lower cold time and 5% lower maximum peak RSS than `b802a76`, but remain rejected because neither side met the 5% CV contract.
9. **Done — harden the benchmark protocol against first-run and host variance.** The harness now performs unmeasured candidate and optional baseline validation, alternates candidate/baseline order by round, verifies file and diagnostic accounting within each engine, and rejects reports when either cold-run CV exceeds the configured threshold. Both post-allocation WordPress attempts were rejected as designed despite favorable candidate means. The remaining task is environmental/statistical stability, not bypassing the gate.

Note: a benchmark run on `test_projects/symfony` also showed the diagnostic count vary slightly between cold runs on an otherwise-identical corpus (e.g. 82,722 vs 82,883 in one sample). **Fixed:** `BuildProjectIndex` iterated its `map[string][]ast.Node` input in Go's randomized map-iteration order, so which file's declaration won duplicate-symbol resolution (`addClass`'s "first file wins", and "last file processed wins" for methods/properties/constants registered per file) varied between runs. It now processes files in sorted filename order, making both the class-metadata winner and the member winner deterministic; `TestBuildProjectIndexDuplicateClassResolutionIsDeterministic` in `analyse/project_index_test.go` guards this, and a 5-run benchmark on `test_projects/symfony` now reports a stable 82,722 diagnostics on every cold run.

Note: fetching and benchmarking `wordpress-develop` (3,188 files) for the first time surfaced two real parser gaps, one now fixed and one still open:

- **Fixed:** the `global $a, $b;` statement had no parser support at all — not even inside functions — and failed with "unexpected token global in expression" everywhere it appeared. Added `ast.GlobalVarDeclNode` and a `T_GLOBAL` case in `parseStatement` (`parser/statement.go`), mirroring the existing `static $x;` handling. Regression tests in `parser/global_inline_html_test.go`.
- **Fixed:** the lexer/parser had zero support for inline HTML — any file mixing PHP and HTML (extremely common in WordPress admin/template files: `<?php ... ?>\n<div>...</div>`) failed outright, and so did any file with literal content before the first `<?php`. Added an `inHTML` scanning mode to the lexer (`lexer/lexer.go`): a `?>` close tag emits `T_CLOSE_TAG`, switches to raw-text scanning, and optionally consumes one trailing newline per PHP semantics; the next `<?php` or `<?=` re-enters PHP-token mode (`<?=` is expanded to an implicit `echo`). Raw text between tags becomes a single `T_INLINE_HTML` token, surfaced as `ast.InlineHTMLNode`. `parseStatement` and the top-of-file expression/echo statement parsers now also accept `?>`/EOF as an implicit statement terminator (PHP allows omitting the `;` right before a close tag). To avoid a large, unrelated blast radius, the lexer's default mode was deliberately left as "PHP code" (not "HTML") at position zero, since ~200 existing lexer/parser tests (and some callers) construct bare PHP snippets without a leading `<?php` and rely on immediate PHP tokenization; only an explicit `?>` seen mid-stream switches into HTML-scanning mode. As a result, content before a file's first `<?php` tag is still unsupported (a pre-existing, unchanged limitation) — the fix targets the much more common mid-file/trailing-HTML pattern. Together with the `global` fix, wordpress-develop's parse-failure rate dropped from 1,248/3,188 files (39%) to 628/3,188 (~20%).
- **Not yet fixed:** the majority of the remaining ~20% of wordpress-develop parse failures are PHP's alternative/colon control-structure syntax (`if (...): ... elseif (...): ... else: ... endif;`, and the `for`/`foreach`/`while`/`switch` equivalents), which is idiomatic in template files and heavily used alongside inline HTML. The lexer already emits `T_ENDIF`/`T_ENDFOR`/`T_ENDFOREACH`/`T_ENDWHILE` tokens but the parser has no alternative-syntax statement handling at all. This is the natural next parser-compatibility target.
- magento2 (25,390 files) had a much lower failure rate throughout (108→87 files after these fixes, ~0.3%) and is otherwise a clean, deterministic benchmark target (169,602 diagnostics, stable across runs, ~1.5s cold).

Note: implemented PHP's alternative/colon control-structure syntax and several other parser gaps it exposed, closing out the note above:

- **Fixed:** alternative (colon) syntax for `if`/`elseif`/`else`/`endif`, `for`/`endfor`, `foreach`/`endforeach`, `while`/`endwhile`, and `switch`/`endswitch` (`do`/`while` deliberately excluded — PHP has no `enddo`). Also fixed: the `endfor`/`endforeach`/`endwhile`/`endswitch` keywords were missing from the lexer's keyword table entirely (only `endif` was mapped), so they always lexed as plain identifiers. New shared helpers in `parser/alt_syntax.go` (`parseAltBody`, `consumeAltTerminator`) parse a colon-delimited body up to a caller-specified set of stop keywords and accept `;`/`?>`/EOF as the closing terminator, mirroring the existing implicit-terminator rule for `echo`/expression statements.
- **Fixed (regression bug the above surfaced):** `parseBlockStatement` (the general `{ ... }` body parser) and the function-body brace-depth loop in `parser/function.go` both stopped on `T_RBRACE`/`T_EOF` as their *only* loop-exit checks, but delegated tag-transition tokens (`T_OPEN_TAG`/`T_CLOSE_TAG`) to `parseStatement`'s own internal retry loop — which jumps straight from an open tag into whatever token follows it (including the very `}`/keyword that should have ended the loop) without ever returning control to the outer loop's stop-condition check. This broke any function/block body containing inline HTML immediately before its closing `}` (e.g. `function f() { ?><style>...</style><?php }`, extremely common in WordPress template-output helpers), and would have broken the new alternative-syntax bodies the same way. Both loops (plus the new `parseAltBody`) now consume `T_OPEN_TAG`/`T_CLOSE_TAG` themselves before checking their stop condition.
- **Fixed:** a single-line `//`/`#` comment on the same source line as a `?>` close tag swallowed the close tag as part of the comment text (`readLineComment`/`readHashComment` in `lexer/comment.go` only stopped at `\n`/EOF). Per PHP semantics a single-line comment is also terminated by `?>`, which is left unconsumed for the next token. This is a common pattern in WordPress template code (`<?php // comment ?>`).
- **Fixed:** `\false`/`\true`/`\null` — the fully-qualified (leading-backslash) form of these builtin constants, used by some libraries (e.g. SimplePie, bundled in wordpress-develop) to avoid relying on unqualified name resolution — were not accepted as name segments after a leading `T_NS_SEPARATOR` in `parseSimpleFQCNOrFunctionCall` (`parser/expression.go`), which only accepted `T_STRING`/`T_STATIC`/`T_SELF`/`T_PARENT`.
- **Fixed:** PHP allows almost all keywords, including visibility modifiers, as property/method names accessed via `->` (e.g. `$this->public`, `$obj->final()`). `isValidMethodNameToken` (`parser/helper.go`) was missing `public`/`private`/`protected`/`abstract`/`final`/`use`.
- **Fixed:** `echo` accepts one or more comma-separated expressions (`echo $a, $b, $c;`), unlike `print` which only accepts a single expression. The common single-expression case still parses to a bare `ast.ExpressionStmt` (unchanged AST shape, so existing consumers are unaffected); multiple comma-separated expressions are wrapped in an `ast.BlockNode` of per-expression `ExpressionStmt`s, which every existing AST walker already knows how to recurse into (avoided introducing a new `EchoNode` type after it broke several analyse-rule and parser tests that pattern-matched on `ExpressionStmt`).
- Net effect: wordpress-develop parse failures dropped from 628/3,188 (~20%) to 63/3,188 (~2%). Of the remaining 63, 9 are the still-unsupported "content before the first `<?php`" case (see the inline-HTML note above); the rest are scattered, low-frequency one-offs (constructor-promoted `readonly` params, `&$var` reference params in some call-site positions, `list()` destructuring as a `foreach` target, dynamic `Class::$name` after `::`, etc.) not related to this task. All 8 corpora re-benchmarked with no regressions; every corpus's failure count held steady or improved further (e.g. magento2 76 failures, down from 78).
- **Fixed (follow-up batch — remaining long-tail gaps):** all of the previously-listed one-off gaps, plus several more surfaced by a fresh full-corpus scan, are now fixed: error-suppression (`@`) wrapping an assignment target, including `@list(...) = ...` and `@$arr['x'][] = ...`/`@$arr['x'] += ...` (`parser/expression.go`); `readonly` used as a plain function name (`T_READONLY` added to `isValidMethodNameToken`, `parser/helper.go`); by-reference function/method returns, `function &name(...)` (`parser/function.go`, `parser/method.go`); dynamic static-property access with a braced or double-dollar name, `self::${$expr}` and `self::$$name` (new `T_DOLLAR_OPEN_CURLY_BRACES` lexer case in `lexDollar`, new `ConstExpr` field on `ast.ClassConstFetchNode`, handling in both static-access parse paths in `parser/expression.go`); standalone variable-variables, `$$name` (new `ast.VariableVariableNode` and `parseSimpleVariableVariable` helper); trait `use A, B { ... }` adaptation blocks (`insteadof`/`as`, with optional visibility and aliasing) — this also surfaced that the lexer never had an `insteadof` keyword mapping at all (`lexer/lexer_keywords.go`), so `insteadof` always lexed as a plain identifier; new `ast.TraitAdaptation` type and `parseTraitAdaptation` in `parser/class.go`; `foreach ($rows as list($a, $b))` destructuring as the value target (`parser/foreach.go`); by-reference array-literal elements pointing at a property or array offset, not just a bare variable, e.g. `[&$this->prop, &$arr[0]]` (`parser/array.go`, now reuses `isValidAssignmentTarget` instead of a hardcoded `*ast.VariableNode` check); scientific-notation float literals (`1.2e1`, `1.7E+308`) and leading-dot float literals (`.8`, `.111`), both previously unlexed (`lexer/lexer.go`); the `>>=`/`<<=` shift-assignment operators, which were entirely absent from tokens, lexer, and parser operator tables (new `T_SL_EQUAL`/`T_SR_EQUAL`, `lexer/lexer_symbol.go`, `parser/operator.go`); a trailing doc-comment with no following statement inside a block or immediately before a `case`/`default` label, which previously caused cascading "unexpected token" errors (`parser/statement.go`); and PHP tag transitions (`?> ... <?php`) inside `switch`/`case` bodies not being explicitly consumed before the loop's stop-condition check, mirroring an already-fixed bug in `parseBlockStatement` (`parser/switch.go`).
- Net effect of this follow-up batch: wordpress-develop parse failures dropped from 63/3,188 (~2%) to 14/3,188 (~0.4%) — all 14 remaining are the still-deliberately-unsupported "content before the first `<?php`" case. All 7 other benchmark corpora re-scanned with no regressions and every corpus's failure count improved: composer-src 18→11, drupal 215→88, laravel 328→276, magento2 76→27, phpunit 190→181, symfony 297→231. Targeted parser unit tests for each fix were added in `parser/longtail_gaps_test.go`.
- **Fixed (broad cross-corpus batch — all 7 non-wordpress corpora diagnosed together):** a full-corpus scan across composer-src, drupal, laravel, magento2, phpunit, and symfony (wordpress-develop was already at its floor) surfaced dozens more gaps, fixed in two passes:
  - PHP's semi-reserved-word list for method/property names (`isValidMethodNameToken`, `parser/helper.go`) was missing ~40 common keywords (`if`, `else`, `extends`, `catch`, `finally`, `class`, `array`, `mixed`, `as`, etc.) — now far more complete, fixing `->if()`, `function catch()`, `Type::mixed()`, `$pool->as()`, and similar.
  - Grouped `use function a, b, c;` / `use A, B;` import lists (`parser/use.go`) and grouped `const A = 1, B = 2;` declarations, including the typed/visibility-modified class-const form (`public const JPEG = 1, PNG = 2;`) and comments between grouped items (`parser/constant.go`, now returns `[]*ast.ConstantNode` from all call sites, wrapped in an `ast.BlockNode` at the top-level-statement callsite when there's more than one).
  - Trailing commas before a closing `)` in `unset($a, $b,)` (`parser/statement.go`).
  - Comments between a try-block's `}` and a following `catch`/`finally` (`parser/try.go`); trait bodies composing another trait via `use OtherTrait;` (`parser/trait.go`); attributes (`#[...]`) inside interface bodies and directly on call arguments, e.g. `foo(#[Closure] fn () => ...)` (`parser/method.go`, `parser/expression.go`).
  - Dynamic static method/constant access via braces, `Class::{$expr}(...)`, distinct from the already-supported `::${expr}` dynamic-property form (`parser/expression.go`).
  - A systemic PHP grammar quirk where assignment (`=`) is allowed as the operand of higher-precedence constructs (`!`, `@`, casts, the ternary else-branch, comparison operators, `include`/`require`, etc.) even though `=` has lower nominal precedence, e.g. `'nfd' === $rule = strtolower($rule)` parses as `'nfd' === ($rule = strtolower($rule))`. Fixed via a new shared `parseUnaryOperandWithAssignmentStealback()` helper (`parser/expression.go`) that "steals back" a simple operand as an assignment's LHS when immediately followed by an assignment operator; replaces 12 previous call sites of the old precedence-100 helper.
  - First-class callable syntax, `expr(...)`, generalized to any callable expression (not just static/instance method names) — `$callback(...)`, `[$obj, 'method'](...)` — via a new `Target` field on `ast.FirstClassCallableNode` and handling in `parseSimpleVariableFunctionCall` (`parser/expression.go`, `ast/ast.go`).
  - By-reference arrow functions, `fn &() => ...` (`parser/expression.go`).
  - Binary integer literals, `0b1010` (new lexer case in `readNumber`, `lexer/lexer.go`) — and, found in the same code path, a **pre-existing correctness bug**: integer literal values were always parsed with `strconv.ParseInt(..., 10, ...)` regardless of the `0x`/`0o`/`0b`/leading-zero-octal prefix the lexer had already normalized the literal to, silently yielding `0` for anything but plain decimal; now uses base `0` for automatic prefix detection (`parser/expression.go`).
  - `true`/`false`/`null` as standalone types (PHP 8.2) accepted in every type-hint position that was missing them: return types, property types, and parameter types (`parser/type_hint.go`, `parser/typehint.go`, `parser/class.go`, `parser/trait.go`, `parser/parameter.go`); a union return type starting with `static`/`self`/`parent` (e.g. `function dayOfYear(): static|int`) was previously short-circuited into treating `static` as the *entire* return type without checking for a following `|`/`&` (`parser/function.go`).
  - Dynamic class instantiation via `new ${$expr}(...)` and namespace-relative `new namespace\Foo(...)` / `namespace\CONST` references in general expression position (`parser/expression.go`, `parser/parser.go`); a doc-comment between `new` and an anonymous `class { ... }` declaration (`parser/expression.go`).
  - `foreach` loop targets that are a property-fetch expression rather than a bare variable, e.g. `foreach ($users as $this->user)` and `foreach ($items as $stub->class => $stub->position)` (`parser/foreach.go`, reuses the existing property/array-index postfix helper).
  - Cast-type keywords are matched case-insensitively, e.g. `(String)`, `(INT)` (PHP casts are case-insensitive) (`parser/expression.go`).
  - The deprecated `<>` not-equal operator, an alias for `!=` (`lexer/lexer_symbol.go`).
  - Net effect: composer-src 11→0, drupal 88→1, laravel 276→94 (of which ~89 are the known "content before `<?php`" Blade-template limitation), magento2 27→0, phpunit 181→2 (both remaining are PHP 8.4 property-hook syntax in interfaces, not yet supported), symfony 231→48 (mostly the same `<?php` limitation plus a handful of PHP 8.4 asymmetric-visibility/property-hook cases, also not yet supported). All corpora re-verified with `go build`/`go vet`/`go test` green throughout. New targeted parser unit tests in `parser/corpus_batch2_gaps_test.go`. Remaining gaps are predominantly either the deliberate `<?php`-prefix limitation or PHP 8.4 asymmetric-visibility/property-hook syntax (`public(set)`, `{ get => ...; }`), which is a materially larger feature to implement and was left as a follow-up.
- **Fixed (batch 3 — closed the PHP 8.4 property-hooks/asymmetric-visibility gap, plus two significant unrelated lexer bugs found along the way):**
  - **Lexer bug (dead code, silently corrupting `::$var` access):** `lexColon`'s `::class`-merging special case (`lexer/lexer_symbol.go`) checked the wrong peek offset and had in fact never correctly merged real `Foo::class` usage — that always fell through to separate `T_DOUBLE_COLON` + `T_CLASS` tokens, which the parser already handles at two call sites. The *only* thing the buggy check ever did was misfire on `::$var`-style static property access whose name happened to match its broken prefix logic (e.g. `static::$class` was silently mis-tokenized). Fixed by deleting the merging special case entirely; `::class` continues to work exactly as it always effectively did, via separate tokens.
  - **Lexer bug (EOF sentinel conflated with a literal NUL byte):** `readChar()` uses `l.char = 0` as its end-of-input sentinel, which is indistinguishable from a real embedded `\x00` byte in the source (valid inside PHP string literals, e.g. Symfony's own null-byte-injection test fixtures). Every `l.char == 0`/`!= 0` EOF check across `lexer/lexer.go`, `lexer/comment.go`, `lexer/heredoc.go` therefore also treated an embedded NUL byte as EOF, truncating parsing at that point. Added a position-based `atEOF()` helper and replaced all such comparisons.
  - **New backtracking primitive:** disambiguating `public(set)` (asymmetric visibility) from a plain visibility modifier followed by a parenthesized property type (e.g. `public (\Traversable&\Countable)|null $x`) needs multi-token lookahead beyond the parser's existing 1-token peek. Added a general-purpose snapshot/restore mechanism (`lexer.State`/`Snapshot()`/`Restore()` in `lexer/lexer.go`, `parserCheckpoint`/`checkpoint()`/`restore()` in `parser/parser.go`) and rewrote `parsePropertyModifier` (`parser/class.go`) to use it; also added `T_LPAREN` to the property-type-hint-start switches (`parser/class.go` x2, `parser/trait.go`) and routed parenthesized property/interface-property types through the existing `parseFullTypeHint` (which already handled nested parens/unions/intersections, previously only used for parameter types).
  - Constructor-promoted properties with two stacked modifiers, `public private(set) string $title` / `public protected(set) string $author` — added a `SetVisibility` field to `ast.ParamNode` and rewrote the parameter-modifier loop (`parser/parameter.go`) to call the same `parsePropertyModifier` used by class properties, in a loop, so any combination of visibility/asymmetric-visibility/`readonly` is accepted.
  - Typed class constants with union types, `const string|int BAR = 'bar';` — the constant-type detector (`parser/constant.go`) only recognized a single type token followed directly by the constant name; now speculatively parses a full (possibly union) type hint via checkpoint/restore and only keeps it if a constant name actually follows.
  - `match` arms with a trailing comma right before `=>` in a multi-condition arm, e.g. `'a', 'b', 'c', => expr,` (`parser/match_parser.go`).
  - Attributes prefixing an expression in more positions: a `return`ed closure (`return #[When(env: 'prod')] function () {...};`), a closure as an array value (`'key' => #[\Closure(...)] function () {...}`), and other general expression positions — fixed with one general change at the top of `parseExpressionWithPrecedence` (`parser/expression.go`) that skips leading `#[...]` attributes before dispatching, rather than needing a fix at every call site. Also fixed attributes directly on an anonymous-class expression, `new #[\AllowDynamicProperties] class {...}` (`parser/expression.go`).
  - PHP 8.4 property hooks in `interface` bodies (`public string $foo { get; }`) and `abstract` property hooks in abstract classes (`abstract public string $bar { get; }`) — neither form has a hook body/expression, just a bare `get;`/`set;` declaration; added that case to `parsePropertyHooks` (`parser/class.go`), and added interface-body property-hook parsing (`parser/method.go`, new `isInterfacePropertyTypeStart`/`parseInterfaceProperty`) since interface bodies previously only recognized methods and constants.
  - Named arguments on an arbitrary callable expression call, `$controller($name, headers: [...])->headers->get(...)` — `parseSimpleVariableFunctionCall` (`parser/expression.go`) only supported positional/unpacked arguments; now also recognizes the `name: value` named-argument form, matching the already-existing support in `parseFunctionCallArguments`.
  - Net effect: phpunit 2→0 (fully closed), symfony 48→24 (of which 23 are the same `<?php`-prefix limitation and 1 is `Config/Tests/Fixtures/ParseError.php`, an intentionally-invalid PHP fixture used by Symfony's own error-handling tests — correctly out of scope), laravel 94→93, composer-src/drupal/magento2 remain at 0. One symfony fixture (`Cache/Traits/ValueWrapper.php`) contains a literal non-UTF8 byte (`class \xa9`) and is a corrupted/mutated test fixture, not a real parser gap — confirmed via byte-level inspection and left unfixed. All corpora re-verified with `go build`/`go vet`/`go test` green throughout. New targeted parser unit tests in `parser/corpus_batch3_gaps_test.go`.
- **Fixed (closed the "content before `<?php`" limitation, the last systemic gap):** the lexer's inline-HTML mode (`inHTML`) already handled HTML *after*/*around* PHP code, but a file's *default* starting mode was always PHP-code, so any file beginning with literal content before its first `<?php`/`<?=` tag (extremely common in template/view files) failed outright. Rather than changing that default globally — ~200 existing lexer/parser tests construct bare snippets (raw operators, expressions) with no leading open tag and rely on immediate PHP tokenization — added a new `lexer.NewFile(input string) *Lexer` constructor that starts in inline-HTML mode unless the input begins with a recognized open tag, and switched every production call site that lexes real file content (`command/command.go`, `command/file_processor.go`, `cmd/benchmark/main.go`, `cmd/compat-metrics/main.go`, `sharedcache/token_cache.go`) to use it instead of `lexer.New`, which keeps its original always-PHP-mode behavior for bare-snippet callers. Also fixed a related latent bug this surfaced: `<?=` as the very first thing in a file (with nothing before it, so the lexer never entered inline-HTML mode) was never recognized as an open tag in default PHP-code-mode symbol lexing — only `<?php` was — so it lexed as `<` `?` `=` and broke; `lexer/lexer_symbol.go`'s `lexLess` now also recognizes `<?=` directly, expanding it to an implicit `T_ECHO` exactly as the inline-HTML path already did. New tests in `lexer/newfile_test.go` and `parser/corpus_batch3_gaps_test.go`.
  - Net effect: wordpress-develop 14→0 (fully closed), laravel 93→2, symfony 24→2 (both of symfony's are the two already-noted out-of-scope fixtures — `ParseError.php`/`ValueWrapper.php` — so this is effectively fully closed for symfony too), composer-src/drupal/magento2/phpunit remain at 0. Laravel's 2 remaining failures are unrelated, narrow vendor-code edge cases: nested string interpolation inside a `"{$...}"` expression that itself contains another interpolated string, and a dynamically-interpolated string literal used directly as a callable name (`"is_$search->mode"(...)`) — both rare enough to leave as follow-ups rather than block this fix. All corpora re-verified with `go build`/`go vet`/`go test` green throughout.
- **M1 progress (complete source spans + structured diagnostics, the first M1 exit-criteria item):** `ast.Node` previously carried only a start `Position` (line/column/offset); there was no end position, so diagnostics and any future editor integration could only report a point, not a range. Added `GetEndPos()/SetEndPos(Position)` to the `Node` interface and an `EndPos Position` field to every one of the ~80 concrete node types (mechanical, scripted edit across `ast/*.go`). Wired the parser to populate it with minimal blast radius: `token.Token` gained an `EndPos()` method (accounts for multi-line literals like heredocs/comments by counting embedded newlines), the `Parser` now tracks `prevTokEnd` (the end of the last consumed token, updated in `nextToken()`), and the two central recursive parsing entry points — `parseStatement` and `parseExpressionWithPrecedence` — were each split into a thin wrapper plus an `*Impl` function, where the wrapper stamps the returned node's `EndPos` to `prevTokEnd` after the impl returns. Because both are the common dispatch point for essentially all statement and expression parsing (including recursive sub-expression calls), this gives full-file span coverage without touching the ~150 individual node-construction call sites throughout the parser. `analyse.AnalysisIssue` gained `EndLine`/`EndColumn` fields (zero means "unknown span", matching the same convention as an unset `EndPos`), and a new `issueSpan(filename, node, code, message)` helper (alongside the pre-existing point-only `issue(filename, pos, code, message)`) was adopted at all 68 (of 87 total) issue call sites that had a concrete node available, covering effectively all of the PHPStan-level-0 rule's diagnostics. New tests: `token/token_test.go` (multi-line `Token.EndPos()`), `parser/span_test.go` (statement- and expression-level span coverage through a real parse). Known pre-existing, out-of-scope quirk noticed along the way (not touched): some nodes' *start* position points at an inner token rather than the construct's true first token (e.g. an assignment's `ExpressionStmt` start position points at `=`, not the left-hand variable) — orthogonal to end-position work, left for a future pass. The immutable semantic snapshot, stable symbol identities, and shared fact-store boundary are now also complete; broader foundational type operations and PHPStan level 1–3 progress remain.
- **M1 progress (first control-flow graph slice):** `SemanticSnapshot` now owns immutable statement-level control-flow graphs keyed by file and exact lexical-scope spans. `FlowGraphReader` exposes defensive graph values, statement reachability, and normal-fallthrough queries through `AnalysisContext`; registered-rule context cloning preserves the reader. `Generic.CodeAnalysis.UnreachableCode` consumes snapshot reachability while retaining its legacy fallback and exact diagnostics when no graph is available. Compound statements remain atomic in parent graphs and have separate child-scope graphs; missing or duplicate spans remain unknown rather than aliasing facts. Adversarial coverage verifies exhaustive and partial branches, nested unreachable code, deterministic IDs, defensive copies, ambiguous spans, and concurrent reads. Full parent/child graph expansion, `try`/`switch`, and cross-scope multi-level transfers remain before the graph can support joined variable-state analysis.
- **M1 progress (return completeness):** `A.RETURN.TYPE` now uses the snapshot's function-scope fallthrough result to report declared non-`void` functions, methods, and closures that can complete without returning a value. Nullable, `mixed`, and `never` declarations retain PHP's explicit-return requirement; abstract declarations and generators are excluded, while `throw`, `exit`, `die`, and exhaustive `if`/`elseif`/`else` paths count as termination. The rule keeps its deterministic legacy termination fallback when a graph scope is unavailable and now discovers functions consistently inside namespaces, traits, enums, and nested expressions. Adversarial tests cover graph authority, missing-scope fallback, nested-generator ownership, runtime-sensitive return types, methods, closures, and terminating branches. Broader `try`/`switch`/loop precision remains coupled to the next CFG expansion.
- **M1 progress (first-class `for` loops):** the parser no longer discards `for` control clauses or disguises the loop body as a generic `BlockNode`. `ast.ForNode` preserves ordered initializer, condition, and update expression lists (including empty clauses), the body, and the complete source span. Analysis, hover, diagnostic, fact, printing, and reachability walkers now traverse the node explicitly; initializers update the enclosing scope while body/update state remains loop-local and conservative. The semantic snapshot creates a distinct `for` body scope; the parent graph remains conservatively atomic while the child graph now models loop back-edges and direct control transfers.
- **M1 progress (loop CFG edges):** `break` and `continue` now have first-class AST nodes with optional preserved levels and complete spans instead of lossy expression wrappers. Loop child graphs model conditional header exits and body back-edges for `for`, `while`, and `foreach`; `do` graphs use a distinct post-body condition block. Direct and numeric level-one `break`/`continue` target the loop exit or continuation point, exhaustive conditional transfers expose both targets, constant boolean conditions refine loop exits, and `for (;;)` falls through only when a reachable break exists. Numeric transfers above level one are retained in the AST but deliberately have no incorrect inner-loop edge until cross-scope targets are represented. All analysis and fact walkers now traverse loop-control levels and `do` bodies/conditions explicitly.

## Decision log

- Go remains the implementation language. The target is considered achievable in Go; architecture, allocation behavior, semantic work, and concurrency are the primary constraints.
- Mago is a performance and capability benchmark, not a dependency and not an authority for PHP semantics.
- Relative same-machine results are authoritative; a stale absolute headline is contextual evidence only.
- Index-only and full-analysis performance will always be reported separately.
- Correctness and semantic coverage are release gates for performance claims.
- The 2026-08-25 WordPress profile promotes allocation-light resolver traversal and persistent branch state to M1 prerequisites; richer rules must not be layered onto repeated whole-collection copying.
- Private source code may expose failures but will not be copied into public fixtures or required for benchmark reproduction.
