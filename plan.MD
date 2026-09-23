# Current implementation plan

Status: approved, not started

Roadmap source: short-term item 2

Scope: reduce false positives and expand executable compatibility

This file is the single source of truth for work currently being implemented. Keep the next implementation within this approved roadmap item; record evidence before accepting historical diagnostic counts as baselines.

## Objective

Reduce false positives across the parser and analyser while expanding neutral executable coverage for supported PHP and framework behavior.

## Work sequence

### 1. Establish reviewed diagnostic baselines

- [ ] Re-run the exact private-corpus manifest before accepting historical diagnostic counts as a baseline.
- [ ] Build reviewed true-positive, false-positive, and false-negative suites with pinned reference versions and configuration.
- [ ] Add neutral public fixtures for each correction.

### 2. Close high-priority type and resolver cases

- [ ] Close remaining `Level6.MissingIterableValueType` cases across native, PHPDoc/refined, inherited, trait, and anonymous-class contracts.
- [ ] Audit repository return/template substitution and unknown-symbol noise.
- [ ] Add framework metadata for Symfony container returns, Doctrine repositories and collections, PHPUnit data providers, Composer, and common Laravel patterns.

### 3. Expand executable PHPStan compatibility

- [ ] Add PHPStan-gated coverage at the level where each diagnostic begins: casts, offset access, callables, increments, foreach, interpolation, condition narrowing, PHPDoc validation, property initialization, variance, and trait/interface contracts.
- [ ] Keep the capability matrix, README rule inventory, and rule pages synchronized with executable registrations.

## Acceptance gates

- Level 0 behavior is demonstrated on the agreed corpus rather than inferred from fixture totals.
- Levels 1–6 have broad executable coverage and explicit unsupported identifiers.
- Framework packs pass without global suppressions.
- Reviewed false-positive and false-negative release thresholds exist and pass reproducibly.
