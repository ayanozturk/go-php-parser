# Current implementation plan

Status: in progress

Roadmap source: short-term item 2 (performance priority explicitly selected by the user)

Scope: implement and measure the profile-backed 2× full-analysis performance plan.

## Objective

Deliver at least 2× faster process-cold full analysis on each pinned PSL, Symfony, WordPress, and Magento workload against equivalent code, preserving complete behavior and workload accounting. Aim for 3× where evidence supports it; keep peak RSS at or below baseline. Composer is a required regression corpus. The semantic/type-system work is paused while this plan is active; its unchecked work remains on the roadmap and is not marked complete.

## Work sequence

1. [ ] Establish release-toolchain, four-corpus cold, profile, and exact issue-set baselines. Record compiler, build flags, corpus manifests, environment, phase data, and resource counters. Add instrumentation needed to see exclusive phases.
2. [ ] P1: compare bounded identifier-cache designs and remove no-op allocations in relative class type mapping. Land each independently only with corpus parity and measured full-workload results.
3. [ ] P2: share semantic inference and other immutable per-function results with complete flow/scope/generation keys; retain extension consumers and flow/path budgets.
4. [ ] P3: prototype compact internal token/trivia storage and remove measured red-node wrapper allocations; preserve public APIs and lossless recovery.
5. [ ] P4: reduce declaration/lowering duplication; consider structural type identities only if profiles still justify them.
6. [ ] P5: tune worker scheduling, Go toolchain, and PGO as separately reported experiments.
7. [ ] Meet per-corpus 2× cold engine and CLI acceptance gates, no peak-RSS regression, exact diagnostics, and Composer non-regression. Keep Mago comparisons separate.

## Active evidence and constraints

- Execution details, targets, gates, and commands: [docs/plan-performance-2x.md](docs/plan-performance-2x.md).
- Initial profiles and PSL/Composer baselines: [docs/benchmarks/2026-10-04-performance-planning.md](docs/benchmarks/2026-10-04-performance-planning.md).
- The initial fresh evidence covers only PSL and Composer; it does not establish the four-corpus baseline or any speedup.
- Preserve all files, vendor declarations, bodies, rules, parser recovery, diagnostics, and analysis budgets. No reductions or path truncation count as an optimization.
- Update this plan and the linked execution record as gates complete. Do not check off performance delivery until acceptance evidence exists.
