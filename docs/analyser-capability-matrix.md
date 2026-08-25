# Analyser Capability Matrix

## Evidence contract

This matrix separates executable evidence from descriptive coverage claims. A capability is not described as PHPStan-compatible merely because a similarly named rule exists.

The first checked-in differential pack lives in `testdata/diagnostic-differential`. Its manifest maps each neutral PHP fixture to the exact diagnostic code expected from this engine and the exact PHPStan error identifier expected from the reference analyser. `cmd/diagnostic-diff` reports unexpected, missing, and duplicate diagnostics because it compares sorted identifier lists rather than only checking whether any error occurred.

Run the local engine gate:

```sh
go run ./cmd/diagnostic-diff --engine-only
```

Run the full differential against an installed PHPStan binary:

```sh
go run ./cmd/diagnostic-diff --phpstan-bin /absolute/path/to/phpstan --json
```

The full report records the PHPStan version returned by the supplied executable. Results from different reference versions must not be merged without review. The ordinary Go suite uses engine-only mode so it does not silently download or depend on an unpinned external analyser.

## Executable differential coverage

| Capability | Status | Fixture evidence | Engine diagnostic | PHPStan identifier | Current boundary |
| --- | --- | --- | --- | --- | --- |
| Unknown instantiated classes | Partial, differential-gated | `unknown-class` | `PHPStan.Level0.Symbols` | `class.notFound` | The analyser covers several class-reference surfaces, but the first pack proves only direct `new` expressions. |
| Unknown function calls | Partial, differential-gated | `unknown-function` | `PHPStan.Level0.Symbols` | `function.notFound` | Built-in and extension-sensitive symbol coverage remains incomplete. |
| Function argument counts | Partial, differential-gated | `argument-count` | `PHPStan.Level0.Invocation` | `arguments.count` | The first pack covers a direct known function call; dynamic calls and constant-array unpacking remain outside the gate. |
| Always undefined variables | Partial, differential-gated | `undefined-variable` | `PHPStan.Level0.Variables` | `variable.undefined` | Branch-sensitive possibly-defined variables require joined-state dataflow over the control-flow graph. |
| Known symbols without false positives | Partial, differential-gated | `clean-known-symbols` | none | none | The clean fixture covers a declared function and compatible call, not the full false-positive surface. |

## Implemented but not yet differential-gated

These areas have repository unit coverage but no checked-in PHPStan differential fixture yet:

| Area | Current status | Principal diagnostic codes |
| --- | --- | --- |
| Class hierarchy and modifier legality | Partial | `PHPStan.Level0.ClassModel` |
| Type, import, catch, and attribute references | Partial | `PHPStan.Level0.Symbols` |
| Constructor, method visibility, named arguments, and static/instance call direction | Partial | `PHPStan.Level0.Invocation` |
| Class constants and property access | Partial | `PHPStan.Level0.Symbols`, `PHPStan.Level0.ClassModel` |
| Selected PHP language legality checks | Partial | `PHPStan.Level0.Language` |
| Return completeness, return types, and property assignment types | Partial, above level 0 | `A.RETURN.TYPE`, `A.PROP.TYPE` |
| Argument types | Partial, above levels 0–3 | `A.ARG.TYPE` |

## Unsupported milestone areas

| Capability | Status | Dependency |
| --- | --- | --- |
| Possibly undefined variables | Not implemented | Control-flow graph and joined scope states |
| Arbitrary-expression unknown method checks | Not implemented | Reusable expression types and broader member resolution |
| PHPDoc validation parity | Not implemented | Complete PHPDoc type validation and source mapping |
| Full level 0 parity | Not implemented | Expand the differential pack across the agreed corpus and close reviewed mismatches |
| Quantified false-positive/false-negative thresholds | Not established | Larger reviewed differential corpus with pinned reference reports |

The broader descriptive inventory remains in `docs/phpstan-levels-0-3-rules-comparison.md`. That document must not be treated as executable parity evidence unless a row is linked to this differential pack.
