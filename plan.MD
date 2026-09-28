# Current implementation plan

Status: complete

Roadmap source: short-term item 2

Scope: reduce false positives and expand executable compatibility

This is the previously reprioritized next item after short-term item 3 was completed and delivered. Begin with the exact private-corpus manifest and current false-positive / false-negative baselines before accepting historical counts.

## Objective

Improve analyser compatibility through evidence-backed semantic fixes, neutral fixtures, and framework metadata while preserving the agreed corpus behavior and keeping capability documentation synchronized with executable rules.

## Baseline evidence (2026-09-28)

- The external HR corpus contains 1,930 PHP files (source manifest SHA256 `e0a6e4771fbdc5853fa9931d64e6c0bed7c5cf35697dd1d8f2880ce9fb647bf2`). Its PHPStan 2.1.44 phar is pinned by SHA256 `f9e9524f8e3d3c6338ced11d1911860989165edeedc5b1084d8d53abd87cc7ab`; the level-7 configuration SHA256 is `2cebafcacff07d42a69719403533ec3bc44905030bb4f7ee61315691080a5526`.
- Reference baseline: PHPStan reported 0 diagnostics across the 1,930 files at the configured level 7. The production-equivalent PHP Strom scan with parser `fbc5ea2a` reported 30 `A.ARG.TYPE` findings across four files and no other reportable diagnostics. On this measured corpus/run, the diagnostic comparison is 0 true positives, 30 false positives, and 0 false negatives.
- The findings reduce to short-ternary `DateTime::createFromFormat()` falsey narrowing and PHPUnit `assertNotFalse()` narrowing. Private source, paths, and messages remain external and are not fixture material.
- Differential suites use their checked-in PHPStan 2.2.5 pin, separate from the private project's PHPStan 2.1.44 reference. Levels 0–8 pass with 98, 26, 105, 44, 3, 41, 30, 10, and 51 cases respectively: 408 neutral cases total, with zero engine or reference mismatches and two explicit unsupported level-0 boundaries.
- After parser commit `a0004f15` was pinned in PHP Strom, the production-equivalent rerun indexed 15,699 files and reported 0 diagnostics on the same external project; all 30 baseline `A.ARG.TYPE` findings are cleared. Against the same PHPStan level-7 zero-diagnostic baseline, this rerun measured 0 true positives, 0 false positives, and 0 false negatives. The private-corpus result and the exact-level differential suites satisfy this plan's acceptance gates.

## Work sequence

### 1. Re-establish baselines

- [x] Re-run the exact private-corpus source/config manifest with its pinned reference version and configuration.
- [x] Record current true-positive, false-positive, and false-negative counts for the measured private run; do not infer these from fixture totals.
- [x] Run all checked-in differential suites and identify the highest-impact remaining gaps.

### 2. Fix compatibility gaps

- [x] Close `Level6.MissingIterableValueType` coverage across native, PHPDoc/refined, inherited, trait, and anonymous-class contracts.
- [x] Audit repository return/template substitution and unknown-symbol noise; the pinned level-2 and level-3 packs cover Doctrine generic repository returns, unknown PHPDoc/type references, and clean controls with no remaining mismatches.
- [x] Add framework metadata fixtures for Symfony container returns, Doctrine repositories and collections, PHPUnit data providers, Composer, and common Laravel patterns.
- [x] Expand PHPStan-gated coverage at the level where each diagnostic begins: casts, offset access, callables, increments, foreach, interpolation, condition narrowing, PHPDoc validation, property initialization, variance, and trait/interface contracts; unsupported readonly-initialization and parameter-variance identifiers remain explicit.

### 3. Stabilize evidence and documentation

- [x] Build reviewed true-positive, false-positive, and false-negative suites with pinned reference versions and configuration; the 408-case release gate requires exact engine and PHPStan diagnostic lists with zero mismatches.
- [x] Keep the capability matrix, README rule inventory, and rule pages synchronized with executable registrations.

## Acceptance gates

- [x] Level 0 behavior is measured on the agreed corpus.
- [x] Levels 1–6 have broad executable coverage and explicit unsupported identifiers.
- [x] Framework compatibility packs pass without global suppressions.
- [x] Reviewed false-positive and false-negative release thresholds exist and pass reproducibly.
- [x] Every diagnostic correction has a failing neutral fixture and a clean control at the reference analyser's exact level.
