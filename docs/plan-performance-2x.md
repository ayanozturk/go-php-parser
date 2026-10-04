# Plan: at least 2× faster full analysis

Status: researched execution plan, requested 2026-10-04; optimizations not yet implemented.

[README](../README.md#full-analyser-benchmark) · [Measurements](benchmarks/2026-10-04-performance-planning.md) · [Opportunity register](plan-performance-throughput.md)

## Goal and scope

Deliver **at least 2× faster process-cold full analysis on each of PSL, Symfony,
WordPress, and Magento**, against a frozen equivalent engine baseline. Aim for
3× where the profiles support it. Composer is an additional regression corpus.
Require no peak-RSS regression on any required corpus. Target half the allocated
bytes; a 25% RSS reduction is a stretch goal. Faster parsing alone does not satisfy
this goal. The existing Mago comparison is a separate acceptance track.

The reference engine for this research is `42f88abb6a1005732901c10d975ed68f12cdea7f`.
If semantic development advances before execution, freeze a new baseline at the
start of optimization and repeat measurement; do not compare engines doing
different semantic work. This document prepares the performance iteration;
[plan.MD](../plan.MD) still owns the active semantic implementation queue.

Preserve every selected file, vendor declarations, function body, registered
rule, parser recovery behavior, diagnostic identity, and existing analysis
budget. No reduced rule levels, new truncation, omitted files, or optimistic
cache invalidation may contribute to the speedup.

## What the current evidence says

Fresh profiles and ten-run baselines cover PSL and Composer only. On this
workspace's two-CPU quota, PSL averages **4.3307 s**, Composer **1.3506 s**;
the corresponding half-time thresholds are **2.16535 s** and **0.67530 s**.
These absolute thresholds apply only to this workload and environment.

- Full parsing accounts for about 21% cumulative sampled CPU on both corpora.
  Tokens, green tokens/nodes, and leading trivia together allocate about 35%
  of sampled bytes. Parsing including descendants allocates about 45%.
- File semantics consumes 19–21% cumulative CPU; inferred facts alone consume
  14–17%. Rule execution consumes 35–37%. These stacks overlap and cannot be
  summed as independent costs.
- Identifier normalization consumes 14.50% cumulative CPU on PSL, but 5.27% on
  Composer. PSL's `sync.Map.dirtyLocked` consumes 8.02% cumulative CPU: cache
  churn is a concrete first candidate, not a universal 14.5% wall-time saving.
- GC scanning consumes substantial CPU. Reducing retained pointers and total
  allocations can help several phases, but GC savings overlap their gains.
- The declaration parser already streams and omits body interiors. The full
  host parse follows the declaration pass; these are not two full parses.
  Composite interner hashing, lowering memoization, and several fused rule
  walks already exist. Reimplementing them is not the route to 2×.

The current compiler is Go 1.23.12, versus Go 1.27.1 in the historical corpus
record. Re-profile the intended release compiler before committing to a cache
redesign: `sync.Map` internals are version-sensitive. Compiler upgrades and PGO
must be reported separately from a same-toolchain source improvement.

## Ordered work packages

The cumulative wall-time ratios below are **required planning milestones, not
predicted results**. Each ratio is relative to the frozen original baseline,
measured on every required corpus. They allocate the reduction needed to meet
the goal; profiles do not prove that these reductions are available. Report
both the preceding-commit and original-baseline comparison after each package.

| Package | Concrete change and deliverable | Cumulative target | Effort / risk |
| --- | --- | --- | --- |
| P0 | Freeze workloads, release compiler, diagnostic oracle, phase/counter instrumentation, four-corpus baseline | 1.00 | Small / low |
| P1 | Remove normalization-cache churn and no-op relative-type allocations | ≤0.88 | Small–medium / medium |
| P2 | Share function semantic results and remove repeated inference/lowering work | ≤0.68 | Large / high |
| P3 | Compact full-parser tokens/trivia and avoid transient red-node allocations | ≤0.50 | Large / high |
| P4 | Reduce declaration/lowering duplication; introduce structural type identities where measured | ≤0.43 | Large / high; reserve for missed targets and headroom |
| P5 | Qualify scheduling, PGO, and further measured structural work | ≤0.333 stretch | Variable / measurement-dependent |

### P0 — make the comparison trustworthy

1. Fetch the pinned four corpora and Composer. Save sorted path/content hashes,
   configuration, corpus and engine SHAs, binary hashes, compiler/build flags,
   CPU model/quota, memory limit, worker count, GOMAXPROCS, GOGC, and GOMEMLIMIT.
2. Run a same-binary interleaved null experiment, then at least ten measured
   process-cold runs per engine, sequentially on one host. Require CV ≤5% for
   each engine/corpus; retain all samples. Aim for ≤2% host noise.
3. Add opt-in instrumentation to `cmd/benchmark/main.go` and the production
   pipeline: exclusive discovery/read, declaration parse/lower, index,
   full parse/lower, semantics, rules, reduction, and CLI persistence/output.
   Concurrent phase durations are not additive wall time. Record CPU and
   allocated bytes alongside wall time, plus peak process-tree RSS.
4. Add counters for parse calls, bytes/tokens/nodes, cache hits/misses/evictions,
   lowering memo hits, expression inference calls, facts, and analyzed bodies.
   Add opt-in mutex/block profiles; the current CLI exposes CPU/heap profiles
   only. Run instrumented profiles separately from acceptance timings.
5. Save exact issue snapshots using `cmd/analysis-corpus-snapshot`, including
   all files and levels. Counts alone are insufficient. Preserve parser-error
   paths, symbol/index fingerprints, and CLI diagnostic order as separate gates.

Exit: all four full workloads complete with valid accounting and stable baseline;
any compiler-dependent profile discrepancy is resolved before choosing P1's cache.

### P1 — two bounded changes with immediate evidence

**Ticket 1: identifier normalization.** In `analyse/ascii.go`, compare the current
bounded `sync.Map`/FIFO cache with bounded sharded mutable maps and a worker- or
snapshot-owned cache. Profile lookup, eviction, lock contention, and retained
keys separately. Prefer canonical identifiers at project-index boundaries when
that removes lookups entirely. Keep the existing no-allocation lowercase fast
path, Unicode fallback, case-insensitive resolution, and explicit byte/entry
limits. Do not grow the cache without a measured memory budget.

**Ticket 2: unchanged relative class types.** In `analyse/types.go`,
`withRelativeClassNames` constructs `newAtoms` and a map before discovering that
nothing changed. Detect the no-op path first or allocate lazily. Preserve DNF,
quoted literals, generic/callable nesting, and declaring versus receiver class
context. Do not share mutable slices across callers.

Land these independently with exact diagnostic parity and per-corpus A/B data.
If normalization ceases to be hot on the release compiler, drop the cache
replacement and retain only measured wins. Exit: ≤0.88 cumulative or a written
miss with a re-profile and updated remaining budget; never infer whole-engine
savings directly from cache CPU percentages.

### P2 — compute semantic answers once per valid context

Targets: `analyse/semantic_snapshot.go`, `rules.go`,
`pre_lowered_index.go`, and `syntax_fused_walk.go`.

- Build a per-function work record with symbol/class context, CFG/flow state,
  inferred types, and existing lowering memo references. Hoist repeated
  function-symbol lookup out of per-expression fact insertion.
- Trace duplicate inference between generated inferred facts, shared argument
  checks, return checks, and assignment consumers. Share immutable answers
  keyed by expression **and flow state**, scope, and snapshot generation.
  Span-only caching is incorrect across branch refinements and loop states.
- Materialize serialized fact/type strings only when required by a consumer;
  avoid repeated DNF rendering. Let the complete enabled rule set declare
  required facts before the pass, including extension consumers.
- Make currently eager pre-lowered maps/facts lazy where consumers demonstrably
  do not need them. Reuse existing memo buckets rather than adding a parallel
  lowering cache. Fuse compatible traversals while retaining statement versus
  expression boundaries and deterministic issue production.

Exit: duplicate-inference counters fall, corpus issue sets and flow/extension
behavior match, and cumulative time reaches ≤0.68 without RSS growth. Maintain
recursion/path budgets; the historical WordPress path explosion is a warning
against expanding state indiscriminately, not permission to drop paths.

### P3 — cut the allocation and pointer graph at its source

Targets: `token/token.go`, `lexer/lexer.go` (`LexAllContext`, trivia collection),
`syntax/parser.go`, `syntax/green.go`, and measured callers in `syntax/red.go`.

1. Prototype compact internal tokens: source offsets/lengths and token kind,
   with trivia ranges into a shared compact buffer. Today's trivia uses whole
   `Token` values carrying strings, positions, and two trivia slices.
2. Adapt public tokens lazily at API boundaries. Avoid retaining entire source
   buffers through tiny long-lived facts. Compare the current `len(src)/8+16`
   capacity heuristic with measured token density before tuning capacity.
3. Prototype bounded-lookahead lexing for the full parser only if its access
   patterns permit it. The declaration tier already streams; preserve that.
4. Replace measured `Children` consumers with the existing allocation-free
   `ForEachChildDesc` where possible. `ForEachChild` still creates wrappers.
   Arena/child-index storage is a later prototype if green allocations remain
   dominant; retain collision checks and required node-identity semantics.

Exit: tokens/trivia/green allocated bytes fall substantially (prototype target:
≥50% for that group), all lossless/recovery gates pass, and cumulative full
analysis reaches ≤0.50. Cover UTF-8, CRLF, trivia, heredoc/nowdoc, interpolation,
malformed input, cancellation, and public API compatibility. Token microbench
wins without full-analysis improvement do not meet the exit gate.

### P4 — reserve structural changes for the remaining gap

- In `syntax/parse_ast.go`, `cmd/benchmark/main.go`, and `command/analyze.go`,
  prototype direct declaration summaries from the CST instead of temporary
  declaration ASTs. Preserve vendor indexing, PHPDoc and declaration semantics.
  Measure removed allocations against adapters added for compatibility.
- Full host parsing still needs bodies. Reuse selected parse artifacts only
  within a bounded lifetime when measured cheaper than reconstruction. Current
  `DropSourceFiles`/`DropFileTypeASTRefs` release index ASTs deliberately;
  retaining every full tree is not the default optimization.
- If type rendering, hashing, and substitution remain hot after P2, prototype
  immutable structural TypeIDs scoped to a project snapshot, with memoized
  operations keyed by full generic/class context. Cover literal types,
  intersections/unions, recursive aliases, and cross-snapshot invalidation.

Exit: ≤0.43 cumulative creates headroom beyond the minimum; reject any variant
that trades speed for unbounded retention or changes inferred results.

### P5 — maximize measured throughput

The current harness sets GOMAXPROCS from `--workers`; decouple these controls
before studying scheduling. Sweep workers and scheduler width around the actual
CPU quota (1, 2, 4, then 8 only where resources justify it), track long-file tails,
and consider bounded queues/size-aware scheduling with deterministic reduction.
Do not change workers between baseline and candidate in the code comparison.

Evaluate PGO with mixed representative training profiles and held-out workloads.
Evaluate the supported release toolchain separately. Publish their independent
and combined effects, including binary size and RSS. These experiments may yield
no gain; they are not assumed to close the gap to 2× or 3×.

## Acceptance and reproducible execution

Build baseline and candidate benchmark and snapshot binaries from separate
checkouts using the same compiler and flags. Retain binaries and hashes with
reports. A detached baseline worktree avoids disturbing implementation:

```sh
git worktree add --detach /tmp/php-perf-baseline 42f88abb6a1005732901c10d975ed68f12cdea7f
# In each checkout, using the selected release Go toolchain:
go build -o /tmp/BASELINE-or-CANDIDATE-benchmark ./cmd/benchmark
go build -o /tmp/BASELINE-or-CANDIDATE-snapshot ./cmd/analysis-corpus-snapshot
# Fetch the manifest pins in the candidate checkout:
go run ./cmd/fetch-test-projects --only psl,composer-src,symfony,wordpress-develop,magento2
```

`BASELINE-or-CANDIDATE` above is a placeholder; give each binary its own name.
Run the following for each corpus, retaining all JSON and stdout/stderr:

```sh
/tmp/BASELINE-snapshot --root test_projects/psl --workers 4 --output /tmp/psl-before.json
/tmp/CANDIDATE-snapshot --root test_projects/psl --workers 4 --baseline /tmp/psl-before.json --output /tmp/psl-after.json
/tmp/CANDIDATE-benchmark --root test_projects/psl --workers 4 \
  --baseline-binary /tmp/BASELINE-benchmark --cold-runs 10 \
  --cold-warmups 1 --extra-cold-runs 0 --max-cv 0.05 --skip-warm \
  --json --output /tmp/psl-ab.json
```

For Symfony add benchmark flag
`--expected-parse-errors src/Symfony/Component/Config/Tests/Fixtures/ParseError.php`.
No other parse failures are accepted. Match the pinned manifest's directory
names. Use absolute corpus paths if invoking binaries from other directories.

Acceptance requires:

- Both CVs ≤5%, mean baseline/candidate ≥2 on **each** required corpus, all
  samples retained, no peak-RSS regression, and zero unexplained issue-set or
  workload differences. Composer must not regress. Record failures and skipped
  work explicitly; no averaging away a slow corpus.
- Report a paired 95% confidence interval for the speed ratio from interleaved
  AB/BA pairs; its lower bound must also reach 2. Predeclare the statistical
  method and larger run count before retrying an inconclusive result, and keep
  prior reports. The current harness CV/count validation does not supply this
  statistical or semantic acceptance automatically.
- Run equivalent production CLI cold analyses with fresh result caches and
  identical output format/configuration. Include checksum, cache persistence,
  filtering, sorting, and report rendering. Publish CLI and engine timings
  separately; full-product 2× requires the CLI boundary to pass too.
- Run relevant package/race gates and lossless corpus identity gates when code
  changes; the snapshot oracle must cover Symfony and Composer as well as all
  target corpora. Update rule docs for any separately authorized rule change.

“Cold” means a fresh process and fresh application caches, not forcibly cold OS
page cache. Warm-loop index/analysis and true incremental edits are separate
reports. Incremental optimization may use content/export hashes, dependency
SCC invalidation, and keys covering config, rule/PHP versions, stubs and extension
schema; none of those savings count toward this cold target.

## Decision rules and delivery record

After each package, record SHAs, corpus manifests, exact issue fingerprints,
files/bytes/bodies, mean/median/CV, paired ratio, RSS, allocated bytes, CPU, phase
and cache counters, and the new profile. Land small independently reversible
changes; keep compatibility adapters until public callers migrate. Profiling
instrumentation stays opt-in and outside measured production execution.

Do not add overlapping CPU percentages or multiply guessed speedups. By
Amdahl's law, making 70% of wall work 3× faster yields only 1.875× overall.
The target requires eliminating at least half of complete elapsed work; P4's
0.43 ratio reserves seven percentage points below the 0.50 threshold.

If P1–P3 miss their budget, record the miss and use new profiles to prioritize
P4 or another measured structural bottleneck. Time-box each prototype to one
work package and discard it if full-workload gains disappear or correctness/RSS
fails. The plan is not proof of achievable 2×: keep the objective open until
all acceptance evidence exists, and report a measured ceiling if further safe
improvements cannot reach it. Do not mark performance delivery complete from
this planning commit.
