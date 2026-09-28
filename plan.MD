# Current implementation plan

Status: ready

Roadmap source: short-term item 2

Scope: reduce false positives and expand executable compatibility

This is the previously reprioritized next item after short-term item 3 was completed and delivered. Begin with the exact private-corpus manifest and current false-positive / false-negative baselines before accepting historical counts.

## Objective

Improve analyser compatibility through evidence-backed semantic fixes, neutral fixtures, and framework metadata while preserving the agreed corpus behavior and keeping capability documentation synchronized with executable rules.

## Work sequence

### 1. Re-establish baselines

- [ ] Locate and run the exact private-corpus manifest with its pinned reference versions and configuration.
- [ ] Record current true-positive, false-positive, and false-negative counts; do not infer these from fixture totals.
- [ ] Run the current checked-in differential suites and identify the highest-impact remaining gaps.

### 2. Fix compatibility gaps

- [ ] Close remaining `Level6.MissingIterableValueType` cases across native, PHPDoc/refined, inherited, trait, and anonymous-class contracts.
- [ ] Audit repository return/template substitution and unknown-symbol noise; add a neutral public fixture and clean control for each correction.
- [ ] Add framework metadata for Symfony container returns, Doctrine repositories and collections, PHPUnit data providers, Composer, and common Laravel patterns.
- [ ] Expand PHPStan-gated coverage at the level where each diagnostic begins: casts, offset access, callables, increments, foreach, interpolation, condition narrowing, PHPDoc validation, property initialization, variance, and trait/interface contracts.

### 3. Stabilize evidence and documentation

- [ ] Build reviewed true-positive, false-positive, and false-negative suites with pinned reference versions and configuration.
- [ ] Keep the capability matrix, README rule inventory, and rule pages synchronized with executable registrations.

## Acceptance gates

- Level 0 behavior is measured on the agreed corpus.
- Levels 1–6 have broad executable coverage and explicit unsupported identifiers.
- Framework compatibility packs pass without global suppressions.
- Reviewed false-positive and false-negative release thresholds exist and pass reproducibly.
- Every diagnostic correction has a failing neutral fixture and a clean control at the reference analyser's exact level.
