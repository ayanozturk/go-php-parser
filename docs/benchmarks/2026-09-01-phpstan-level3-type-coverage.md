# PHPStan level-3 type coverage and performance guard, 2026-09-01

## Outcome

The level-3 differential pack expands from one to nine reviewed cases. Existing return- and property-type analysis is now enabled at PHPStan's cumulative level 3 instead of only at the project's former catch-all level 10. The complete executable gates are 88 / 24 / 65 / 9 / 5 / 5 for levels 0–3, 7, and 8.

The production default already executed both type-analysis passes before this change. The implementation changes only their minimum level and category metadata, so it adds no traversal to default PHP Strom analysis. An exact-baseline WordPress cold comparison preserved file and diagnostic accounting but failed both CV gates; no timing or RSS regression/improvement claim is made.

## PHPStan reference evidence

- Parser commit: `4692a1696ef199cc68a14a5e63f36184fe996b60`.
- Reference: PHPStan 2.2.5 from the pinned WordPress dependency tree.
- New matched identifiers: `return.type`, `return.missing`, and `assign.propertyType`.
- New failing cases: mismatched function and method returns, non-exhaustive declared return, and typed-property assignments through `$this` and a typed parameter.
- New clean controls: valid integer and nullable-null returns and a valid typed-property assignment.
- Existing `throw.notThrowable` coverage remains in the same level-3 pack.
- Full differential result: 88 / 24 / 65 / 9 / 5 / 5 with zero engine or reference mismatches.

## Performance guard

The exact previous parser `32d3f28` and candidate alternated over the production WordPress workload with all registered rules, eight workers, `src`, `tests`, and `vendor`, excluding only `src/js`. The harness exhausted its ten extra-run budget because the host changed speed materially during the run.

| Engine | Runs | Mean | Median | CV | Max peak RSS | Accounting |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate | 20 | 2.445s | 2.424s | 13.85% | 1,028,243,456 bytes | 5,357/5,357 files; 22,387 diagnostics |
| Baseline `32d3f28` | 20 | 2.382s | 2.466s | 8.52% | 1,044,561,920 bytes | 5,357/5,357 files; 22,387 diagnostics |

Both CVs exceed the 5% contract, so the raw 2.7% mean difference and 1.6% maximum-RSS difference are rejected. Generated JSON and heap profiles remain under `/tmp` and are not committed.

## Decision

Retain the level correction and expanded differential evidence. It makes existing analysis available at the matching PHPStan level without adding work to the production default. Continue correctness rule by rule, preserve exact corpus/diagnostic accounting, and keep the accepted Mago resource envelope as a guardrail. Do not use this rejected cold run as evidence of either a performance improvement or regression.
