# go-php-parser roadmap

This is the single source of truth for future work in `go-php-parser`. The item currently being implemented lives in `plan.MD` and is not duplicated here as an execution checklist. Detailed capability inventories, rule documentation, architecture constraints, and benchmark results live under `docs/` and in `AGENTS.md`.

## Short term

### 2. Reduce false positives and expand executable compatibility

- Re-run the exact private-corpus manifest before accepting historical diagnostic counts as a baseline.
- Close remaining `Level6.MissingIterableValueType` cases across native, PHPDoc/refined, inherited, trait, and anonymous-class contracts.
- Audit repository return/template substitution and unknown-symbol noise, adding neutral public fixtures for every correction.
- Add framework metadata for Symfony container returns, Doctrine repositories and collections, PHPUnit data providers, Composer, and common Laravel patterns.
- Expand PHPStan-gated coverage at the level where each diagnostic begins: casts, offset access, callables, increments, foreach, interpolation, condition narrowing, PHPDoc validation, property initialization, variance, and trait/interface contracts.
- Build reviewed true-positive, false-positive, and false-negative suites with pinned reference versions and configuration.
- Keep the capability matrix, README rule inventory, and rule pages synchronized with executable registrations.

Done when:

- level 0 behavior is demonstrated on the agreed corpus rather than inferred from fixture totals;
- levels 1–6 have broad executable coverage and explicit unsupported identifiers;
- framework packs pass without global suppressions;
- reviewed false-positive and false-negative release thresholds exist and pass reproducibly.

### 3. Unify diagnostics and finish source mapping

- Use one structured diagnostic schema for parser, style, analysis, and dependency-guard output.
- Complete byte spans and tested UTF-16 conversion for every diagnostic producer.
- Finish style-rule range migration from point locations to exact spans.
- Preserve deterministic ordering and stable machine-readable codes across CLI and editor paths.

Done when:

- no production diagnostic relies on an unstructured message or point-only location where a span is available;
- CLI and PHP Strom publish equivalent codes, messages, and source ranges;
- range, ordering, and malformed-input integration tests pass.

## Medium term

### 4. Complete the semantic and type system

- Complete PHPDoc templates, bounds, variance, generic inheritance, aliases/imports, conditional types, indexed access, key/value projections, callable signatures, shapes, non-empty types, literal types, and integer ranges.
- Complete normalization, subtyping, substitution, narrowing/widening, recursion limits, and `self`/`static`/`parent`/`$this` behavior.
- Extend CFG and variable/property flow through loops, exceptions, generators, closures, references, property initialization, readonly constraints, and dynamic properties.
- Add interprocedural summaries and invalidate dependants only when exported semantic facts change.
- Define testable extension contracts for dynamic returns, assertions, generated members, reflection, service containers, and ORMs.

Done when:

- generic, shape, callable, flow, property, trait/interface, and summary suites meet documented gates;
- Symfony, Doctrine, Composer, PHPUnit, and Laravel compatibility packs pass;
- WordPress, Symfony, PSL, and Magento complete without panic, timeout, race, or silent omission.

### 5. Complete incremental project analysis

- Separate content hashes from exported-semantic hashes.
- Reanalyse only files affected by changed exports.
- Cache parsed snapshots, declarations, type summaries, and diagnostics by content and configuration fingerprint.
- Bound cache memory and expose hit, invalidation, and eviction metrics.
- Cancel obsolete editor work promptly and prevent stale diagnostic publication.
- Partition dependency work deterministically when strongly connected components require ordering.

Done when:

- local edits avoid unrelated project recomputation;
- exported changes invalidate the correct dependency slice;
- cache and cancellation behavior is observable, bounded, and covered by integration tests;
- concurrency cannot change diagnostic output.

### 6. Finish CLI and PHP Strom adoption features

- Add `lint`, `format`, `config`, and `guard` without breaking `style`.
- Add `go-phpcs.yaml`/`.yml` discovery, `--config`, and resolved-config output.
- Add adoption baselines with stale-entry warnings.
- Support text, JSON, GitHub, and Checkstyle output through the shared diagnostic schema.
- Keep autofix opt-in, refuse formatting on parse errors, and apply fixers in stable rule-code order.
- Rewrite PHP Strom feature documentation and remove or clearly mark unsupported settings and commands.
- Replace conservative lexical dependency invalidation after generated reference facts cover supported resolver paths.

Done when:

- CLI commands share configuration and diagnostic semantics;
- user-facing documentation describes only shipped behavior;
- PHP Strom integration tests cover overlays, scheduling, cancellation, invalidation, and UTF-16 conversion.

## Long term

### 7. Meet editor latency targets

- Add a trace-based benchmark for open-document diagnostics, local edits, exported-symbol edits, cancellation, and competing language features.
- Reserve interactive capacity while background analysis runs.
- Prevent stale work from delaying completion, hover, definition, and signature help.

Targets:

- open-document and local-edit diagnostics: p95 at most 100 ms;
- exported-symbol dependency slice: p95 at most 300 ms;
- cancellation acknowledgement: p95 at most 25 ms;
- no stale diagnostics after a newer document version is accepted.

### 8. Reach comparable full-analysis performance

- Profile structural hot paths only after semantic workload and corpus accounting are stable.
- Keep tokens, nodes, and semantic facts compact; intern normalized identities; bound cache lifetimes; and use deterministic parallel reduction.
- Benchmark WordPress, Symfony, PSL, and Magento against a contemporaneous Mago version wherever both tools complete.
- Use process-cold, interleaved runs with identical workloads, complete file/diagnostic accounting, peak RSS, allocations, and CV at most 5%.
- Publish raw machine-readable evidence from a stable host or CI.

Done when:

- cold full analysis is at most 1.5 times the contemporaneous Mago mean on comparable required workloads;
- reliability, semantic coverage, peak RSS, variance, and file-accounting gates all pass;
- no result depends on excluding vendor, reducing rules, skipping bodies, or comparing index-only work with full analysis.

### 9. Full-analyser release

- Establish and meet quantified diagnostic-quality thresholds on reviewed corpora.
- Classify every remaining unsupported language or PHPDoc construct.
- Pass cross-platform builds, race tests, fuzzing, corpus suites, differential suites, editor integration, and reproducible performance gates.
- Publish rule coverage, known limitations, benchmark evidence, and migration guidance.

Done when the parser, analyser, CLI, and PHP Strom meet the correctness, reliability, incremental, editor-latency, and comparable-performance gates above as one shipped system.

## Rules for roadmap work

- Correctness and complete file accounting precede optimization.
- Every diagnostic correction gets a failing fixture and a clean control at the reference analyser's exact level.
- Private source remains private diagnostic input; committed tests use neutral synthetic examples.
- Pure refactors require zero corpus issue-set differences.
- Performance claims require matching workloads and CV at most 5%.
- A parser version is not delivered through PHP Strom until the engine commit is pushed and pinned-module plus sibling-development validation passes.
- Whole-repository line coverage is not a goal; lossless identity, gold/token contracts, high coverage of new kernel code, and behavioral analysis coverage are the gates.
