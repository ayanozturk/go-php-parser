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

The published result was last refreshed on 2026-04-15. The 1.46-second entry is for Mago 1.20.1 even though the current documentation is newer. The public methodology does not identify the benchmark CPU and memory configuration. Absolute times must therefore be reproduced on the same machine before making a comparative claim.

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

### M1 — Complete semantic foundation

Exit criteria:

- Complete source spans and structured diagnostics.
- Immutable project graph and stable symbol identities.
- Control-flow graph, reusable semantic-fact store, and foundational type operations.
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

1. ~~Add the checked-in cold/full/incremental benchmark command and result schema.~~ Done: `cmd/benchmark` (`go run ./cmd/benchmark --root <dir> --json`) measures index-only, process-cold full analysis (subprocess-per-run), and warm-loop full analysis, with mean/median/min/max/stddev/CV, file accounting, diagnostic counts, and RSS (OS rusage + in-process `runtime.MemStats.Sys`). Incremental timing is explicitly reported unsupported pending an incremental-invalidation API (M1/M2). Not yet done: pinning it to the exact Mago benchmark projects/config (action 2) or wiring it into CI as a scheduled job (see Verification and release gates).
2. ~~Pin and automate the Mago benchmark projects and configuration.~~ Done, with an unplanned but higher-priority fix along the way: `test_projects/{composer-src,drupal,laravel,phpunit,symfony}` were committed as bare git "gitlinks" (mode `160000`, like submodule entries) with no accompanying `.gitmodules`, so a fresh clone of this repository silently produced five **empty** directories — every benchmark/compat-metrics claim referencing them (including the determinism and profiling fixes recorded below) was only reproducible on checkouts that happened to already have those nested `.git` directories on disk. Fixed by untracking the gitlinks (`git rm --cached`), ignoring `test_projects/*` except a new `test_projects/manifest.json`, and adding `cmd/fetch-test-projects` (`go run ./cmd/fetch-test-projects`), which fetches each project's exact pinned commit (a single-commit shallow fetch, not a full clone) into `test_projects/<name>` and is idempotent (skips projects already at the pinned commit). The manifest now records exact commits for the existing five corpora plus the three workloads the Mago benchmark itself requires: `php-standard-library/php-standard-library` (pinned to tag `6.2.1`), `WordPress/wordpress-develop` (tag `7.1.0`), and `magento/magento2` (tag `2.4.9`) — the latest stable release of each as of 2026-08-24. Note Mago's own benchmark suite (`carthage-software/php-toolchain-benchmarks`) does not itself pin exact commits for these three projects or publish them in its results feed; it only pins `php-standard-library` via a composer semver range and otherwise tracks each project's default branch, pinning only tool versions. This project's manifest pins exact commits for all eight corpora so every run here is independently reproducible regardless of upstream drift. `wordpress-develop` and `magento2` have now been fetched and benchmarked (see the note below on the two parser gaps this surfaced). Not yet done: CI has not yet been wired to call `cmd/fetch-test-projects` before benchmark/compat-metrics jobs.
3. ~~Minimize and fix the large-corpus project-index panic before further performance tuning.~~ Fixed the reproduced stack-overflow variant of this defect: six ancestor-walking helpers recursed over `extends`/`implements` chains with no cycle detection, so a self-referential or mutually cyclic class hierarchy caused unbounded recursion. All six now carry `seen`-set guards, with a regression test (`TestLevel0CyclicClassHierarchyDoesNotHang`). `cmd/benchmark` exercised this exact code path against the full `test_projects/symfony` corpus (10,478 files, 1.8M LOC) with no crash. The original report described a nil-pointer panic specifically; that signature has not been independently reproduced, so treat this as fixing the reproduced crash class rather than a confirmed root-cause match until re-verified on the original 4M-LOC corpus.
4. ~~Profile allocations, GC, parsing duplication, project-index construction, and per-rule AST walks.~~ In progress: `cmd/benchmark` now supports `--cpuprofile`/`--memprofile`/`--profile-iterations`, running the real parse+index+analyse pipeline in-process (no subprocess boundary) so `go tool pprof` sees genuine allocation and CPU data. A pass against `test_projects/symfony` (10,478 files) found: (a) `PHPStanLevel0Rule.checkLanguage`/`checkSymbolsAndCalls` dominate CPU and allocations (~19% of total CPU, ~1.5GB combined allocated over 5 iterations) as the single largest rule by far; (b) `functionScope.clone` (`return_type_rule.go`) allocates ~580MB by rebuilding two full maps (`variables`, `properties`) per scope split for flow-sensitive narrowing — a candidate for copy-on-write or persistent-map representation; (c) **fixed** a lines-cache thrashing bug: `sharedcache.SplitLinesCached`'s eviction threshold (previously 10,000) tracked cumulative stores rather than live entries and was smaller than several of this project's own benchmark corpora (e.g. `test_projects/drupal` at 10,856 files), so the cache cleared itself mid-run and forced repeated re-splitting of the same files (565MB / 10.5% of total allocations in the profiled run). The counter now tracks live entries (decremented on individual eviction) and the threshold was raised to 200,000 as a memory safety valve rather than a per-corpus budget; total allocations on the same symfony profiling run dropped from 5,361MB to 4,938MB (-8%) with `SplitLinesCached` no longer appearing in the top allocators. Remaining items ((a) and (b) above) are structural rule/type-analysis work, not quick fixes, and are left as follow-ups.
5. Design the immutable semantic snapshot and shared fact-store interfaces.
6. Establish the first diagnostic differential suite and publish a capability matrix.
7. Rebaseline on the same machine against the current Mago release before setting milestone dates.

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

## Decision log

- Go remains the implementation language. The target is considered achievable in Go; architecture, allocation behavior, semantic work, and concurrency are the primary constraints.
- Mago is a performance and capability benchmark, not a dependency and not an authority for PHP semantics.
- Relative same-machine results are authoritative; a stale absolute headline is contextual evidence only.
- Index-only and full-analysis performance will always be reported separately.
- Correctness and semantic coverage are release gates for performance claims.
- Private source code may expose failures but will not be copied into public fixtures or required for benchmark reproduction.
