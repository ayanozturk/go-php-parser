# Consolidated parser, indexer, and analyser performance opportunities

Status: proposal register; not the active implementation plan.

Scope: `go-php-parser` lexer, syntax/CST, lowering, semantic analyser, and
project index, plus the indexer/editor integration in the sibling
`vscode-php-strom` repository.

This document consolidates performance ideas found in both repositories and
removes repeated proposals. It is a research backlog, not authorization to
start optimization work. The current `plan.MD` remains the active work queue;
finish its correctness, recovery, robustness, and accepted baseline gates
before promoting an item here.

## Evidence rules

- Treat source observations from the sibling plans as hypotheses. Several
  proposals are older than the active implementations and need code/profile
  verification before work starts.
- The 2026-09-20 syntax corpus record proves identity/accounting for its pinned
  revision; it is not an accepted cold performance baseline.
- Historical profiles identify candidates, not current hot spots. Re-profile
  the current revision under matching workload and accounting before choosing
  a change.
- The most recent quick syntax loop was directional and over the 5% CV limit.
  Do not use its times as acceptance evidence.
- Performance improvements must preserve source identity, parser recovery,
  deterministic symbol/index results, and analyzer issue sets unless a
  separately reviewed behavior change is intended.

## Consolidated opportunity register

### 0. Measurement and Mago comparison

1. **Establish a stable baseline before tuning.** Finish the active plan's
   phase-separated `ParseCST`, `LowerAST`, and `ParseAndLowerAST` measurements;
   add index-only and full-analysis phase timings for discovery/read, parsing,
   lowering, project-index construction, semantic snapshot, rules, and result
   reduction. Record revisions, machine/runtime, files/bytes/tokens, failures,
   issue/index fingerprints, allocations, RSS, workers, and CV.
2. **Make the workload explicit.** Emit a machine-readable selected-file
   manifest for each tool. Pin corpus revisions, paths, exclusions, PHP and
   tool versions, rule/config boundaries, and cold/warm state. Report file
   accounting and diagnostic breadth beside time.
3. **Measure the noise floor.** Run candidate against the same binary in an
   interleaved null experiment on the proposed host. Use CV <=2% as a host
   qualification target from the sibling proposal; if that proves too strict
   to reproduce, revise it with recorded evidence before deciding performance
   gates.
   Use the new pinned, resource-limited Docker runner to stabilize the Go
   toolchain and CPU quota where useful. A container quota does not reserve
   cores or remove host contention, thermal throttling, or storage pressure;
   retain the CV gate and label shared-host results accordingly.
4. **Harden comparisons where useful.** Verify process-tree RSS for both
   tools; consider paired statistics for interleaved runs and a per-tool file
   manifest. Keep the existing full-sample CV gate at <=5%. Do not let an
   additional statistic overrule failed workload/accounting gates.
5. **Use distinct comparison tracks.** Compare parser throughput only through
   a common parse operation or clearly labelled files/bytes/tokens per second.
   Compare index-only work only against an equivalent declaration/index task.
   Compare full analyzer cold and warm runs separately. Do not label partial
   semantic work as a Mago speed win.

**Target policy:** source plans propose incompatible numerical goals (40-50%
subsystem cuts, 2x parser throughput, <=0.75x Mago time, and <=1.0x Mago RSS),
while the current parser roadmap uses a <=1.5x full-analysis target. None of
these extra thresholds is adopted here. Ratify goals only after a trustworthy
current baseline and comparable Mago workload exist.

### 1. Lexer throughput and storage

1. **ASCII scanning and position bookkeeping.** Profile the current byte/rune
   loop first. If decoding or line/column branches remain material, compare a
   byte fast path with a UTF-8 fallback and a line-start cursor. Preserve
   offsets and positions on mixed UTF-8, LF, and CRLF inputs.
2. **Literal allocation.** Single-character punctuation already uses
   `asciiStrings`; do not duplicate that optimization. Profile remaining
   operator, identifier, numeric, and string allocations. A fixed operator
   literal table or keyword/identifier reuse is worth testing only if it
   reduces end-to-end allocation/time without a larger retained intern table.
3. **Trivia representation.** Measure per-token trivia slice cost. Compare
   inline small-trivia storage or safe scratch reuse against the current
   ownership/copy contract. Do not return pooled slices that callers retain.
4. **Token capacity.** Measure token-count/source-byte distributions and
   `growslice` before changing `LexAllContext` capacity. Avoid paying for dense
   files by overallocating every small file.
5. **Encapsed strings and heredoc/nowdoc.** Benchmark constant and interpolated
   strings at multiple sizes; verify linear work, snapshot/restore frequency,
   and allocations. Retest cancellation inside long scanners.
6. **Compact token text.** Evaluate offset-backed text or compact spans only
   with explicit source-buffer lifetime measurements; avoid retaining whole
   files through a small long-lived token.

### 2. CST construction and traversal

1. **Green interning keys.** Replace formatted string keys for tokens or
   composites with comparable fixed-size/hash keys only if a current profile
   confirms key construction is material. Hashes must be collision-checked
   against complete token/trivia or child identity; preserve interning
   equivalence and distinct literal behavior.
2. **Red child traversal.** Audit hot `.Children()` users and prefer existing
   allocation-free iterators where semantics permit. Compare cached child
   offsets against their retained-memory cost before changing `GreenNode`.
3. **Walk stack.** Avoid needless child reversal; consider reusable/caller-owned
   scratch storage only if walks are hot. Bound pooled capacity so one huge
   syntax tree does not pin a large stack indefinitely.
4. **Position lookup.** If per-node mutex traffic appears in the fresh profile,
   compare a single-goroutine position cursor with the concurrent-safe cache.
   Assert byte-for-byte position equality against the existing path.
5. **Parser dispatch.** Test a single token fetch in `tryParseStructured` only
   if token accessor/branch cost is visible; preserve switch precedence and
   lookahead behavior.
6. **Avoid premature cache growth.** For every cache or interner optimization,
   measure allocations, retained heap/RSS, corpus lifetime, and small-file
   latency. Lower allocation count alone is insufficient.

### 3. Lowering and analyzer work

1. **Parse sharing and declaration/full-body split.** Instrument real call
   paths and `ParseInvocationCount`. The batch benchmark already builds a
   declaration-tier index, releases ingest trees, then does one full
   `ParseAndLower` per host file for semantics and rules. Editor/indexer flows
   may have separate lifetimes; establish whether a file is actually parsed
   redundantly before proposing to derive both tiers from one CST. Retaining
   full-body trees for the whole workspace can cost more RSS than it saves.
2. **Lowering memoization.** Record hit/miss/bridge rates by memo bucket.
   Extend memo lifetime to `ParseResult`/file scope only when hit rate and
   cross-rule reuse justify retention.
3. **Pre-lowered maps and fused-walk state.** Measure whether eager
   `preLoweredIndex` span maps are used. Test lazy construction per map. Profile
   the three fused-walk suppression/callee maps; consider combining state or
   reusing per-file scratch after proving no aliasing across concurrent files.
4. **Semantic snapshot and facts.** Split CFG, variable/property flow,
   inferred-type generation, and fact insertion costs. Consider lazily building
   inferred types only for consumers that need them and reusing immutable
   per-file results when content/config fingerprints are unchanged.
5. **Rule hot paths.** Rank functions by measured CPU and allocations, then
   address repeated PHPDoc parsing, type-string normalization, formatting,
   resolver queries, scope cloning, and control-flow maps only where they are
   top contributors. Rule file size is not evidence of runtime cost.
6. **CST-native rule work.** Port file-wide AST-only work to shared CST walks or
   semantic facts only when the AST compatibility path is measured as costly.
   Preserve issue codes, locations, rule coverage, and shared-walker parity;
   validate corpus snapshots after each cohesive batch.
7. **Caches.** Measure production use and load/save time for the gob AST cache
   and JSON project-index cache before replacing or deleting either. If a cache
   is hot, compare a compact versioned representation and content/config
   fingerprints; preserve invalidation and corruption handling.

### 4. Vendor and project index representation

1. **Keep vendor declarations, skip vendor body analysis.** The current design
   already parses declaration-tier ASTs with method/function bodies omitted,
   indexes those symbols for host resolution, excludes vendor files from
   per-file rules, and drops vendor parse trees before host analysis. Do not
   exclude vendor wholesale: host analysis needs dependency signatures.
2. **Compact vendor signature catalog.** Measure the cost of lowering and
   retaining declaration ASTs after project-index construction. Compare a
   compact resolver-facing declaration summary, ideally extracted directly
   from the CST, while preserving class/function/method/property/constant
   signatures, PHPDoc generics, defaults that affect inference, attributes,
   inheritance, interfaces, traits, visibility, static/by-ref/variadic flags,
   and duplicate resolution. Validate host resolver output against the current
   index before replacing its representation.
3. **Declaration summaries for other unchanged files.** Compare summary
   storage to retained ASTs for full rebuild and incremental update cost; do
   not assume the smallest structure yields the fastest updates.
4. **Incremental fallback precision.** Count full rebuild fallbacks due to
   collisions or incomplete metadata on realistic edit streams. Partition
   affected collision/dependency sets or derive missing metadata only if
   equivalence to a deterministic full rebuild remains exact.
5. **Project index allocation.** Profile sorted-file processing, map copies,
   resolver views, and global compaction. Preserve deterministic winners,
   duplicate reporting, and complete dependency-change metadata.

### 5. Workspace discovery, caches, and interactive scheduling

1. **Worker sweep.** Re-measure indexing at several worker counts (including
   above the current cap where hardware permits), separating I/O and parsing
   from diagnostics. The sibling's 32-worker result is hardware-specific, not
   a default-setting recommendation. Consider work stealing if large-file
   tails leave workers idle. Track throughput, p95 latency, RSS, and editor
   contention.
2. **Discovery and bounded reads.** Time traversal, ignore/exclude matching,
   `stat`, URI conversion, and bounded file reads separately. Check whether
   `DirEntry` metadata can avoid extra stats and whether directory batching or
   queue backpressure helps without changing symlink, ignore, or max-size
   behavior.
3. **Shared line/content caches.** Measure hit rates, allocations, lock cost,
   and memory retention. Reassess pointer-based keys and interface-heavy
   `sync.Map` only against the current implementation. Compare sharded maps,
   content hashes, and bounded eviction; hashing cost and stale-pointer
   correctness must be included.
4. **Incremental semantic invalidation.** Separate content hashes from
   exported-semantic hashes; cache snapshots/declarations/type summaries and
   diagnostics only under complete content/config fingerprints. Measure
   invalidation fan-out and ensure exported changes invalidate the right
   dependency slice.
5. **Editor scheduling.** Measure open-document, local-edit, exported-symbol
   edit, cancellation, and background-index contention traces. Tune debounce,
   worker limits, queue priority, and progress cadence against roadmap p95
   targets. Interactive and batch worker settings may differ.
6. **Cancellation and publication.** Cancel obsolete work promptly; verify no
   partial shared state or stale diagnostics can be published after a newer
   document version.

## Deduplication and source crosswalk

| Unique topic | Repeated sources merged here | Resolution |
| --- | --- | --- |
| Cold/warm benchmark, Mago parity, accounting, CV, RSS | Strom `plan-benchmark-and-mago-parity.md`; Strom `plan-parser-performance.md`; parser `plan.MD` and `benchmark-container/`; parser benchmark records | One benchmark protocol above; container CPU quotas help standardize conditions but do not remove host noise; conflicting numerical targets are explicitly unratified. |
| Lexer ASCII, literal, trivia, capacity, string scans | Strom `plan-lexer-throughput.md`; Strom master plan | Merged into six lexer experiments; punctuation table marked already implemented. |
| Interner keys, empty trivia, red children, Walk, positions, dispatch | Strom `plan-cst-parse-throughput.md`; Strom master plan; parser throughput plan | Merged by subsystem with collision, memory, and profile gates. |
| Memoization, pre-lowered indexes, fused maps, rule profiling, file semantics | Strom `plan-lowering-and-analysis-throughput.md`; Strom master plan; parser analyser plan | Merged as analyzer work; profile before rule ports or cache expansion. |
| Double parsing / parse sharing | Strom indexer and lowering plans; parser analyser and indexer plans; active benchmark architecture | One experiment that first instruments actual lifetimes; the existing batch path is documented accurately. |
| Worker tuning, incremental rebuilds, shared caches, editor latency, discovery | Strom indexer plan/master; parser indexer plan; parser roadmap | Merged into a single indexer sequence; no hard-coded 40% or sub-100ms implementation promise is assumed. |
| Vendor AST retention | Parser indexer plan and current analyser behavior; production scan configuration | Explicitly retained as signatures, not analyzed bodies; compact signature catalog is a distinct follow-up. |
| Gob/JSON cache formats | Strom lowering plan/master | Measure whether they are on the relevant production path before replacing/removing. |
| Semantic facts, identifier folding, resolver objects, scope/CFG allocations | Parser benchmark records (2026-08-31/09-01); Strom analyser plan | Add as fresh-profile candidates; prior fixes mean old hotspots may no longer be current. |

The imported sibling plans also contain numerical targets (40-50% subsystem
cuts, 2x parser throughput, <=0.75x Mago) and claims about exact hotspots.
These are preserved as source ideas in this crosswalk but are not treated as
validated requirements or current facts. Reconcile them only after the
baseline and profiles are accepted.

## Recommended execution order

1. Finish active `plan.MD` and capture the accepted parser baseline.
2. Establish reproducible parser, project-indexer, editor, and full-analyzer
   phase measurements with output fingerprints.
3. Profile the current revision. Remove confirmed duplicate work and excess
   retention first; test the compact vendor signature catalog as a separate
   indexed-resolution experiment.
4. Work from lexer/CST allocation and traversal profiles, then lowerer and
   semantic-rule profiles; accept one change at a time.
5. Tune worker scheduling and caches after serial work and retained state are
   understood.
6. Re-run identity, corpus issue snapshots, deterministic index comparisons,
   editor traces, RSS, and interleaved cold benchmarks before delivery.

## Source plans consolidated

From `vscode-php-strom/docs/`:

- `plan-parser-performance.md`
- `plan-benchmark-and-mago-parity.md`
- `plan-lexer-throughput.md`
- `plan-cst-parse-throughput.md`
- `plan-lowering-and-analysis-throughput.md`
- `plan-indexer-and-incremental-throughput.md`

From the parser repo, the three preceding proposal files have been folded into
this document: `plan-parser-throughput.md`, `plan-indexer-throughput.md`, and
`plan-analyser-throughput.md`. Parser `roadmap.md`, `AGENTS.md`, active
`plan.MD`, and performance benchmark records were consulted for constraints
and prior evidence; they remain separate authoritative sources for their
respective purposes.
