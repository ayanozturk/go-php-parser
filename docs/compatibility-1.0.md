# Tusk 1.0 compatibility contract

This page records the compatibility scope intended for Tusk 1.0. It describes
the supported build targets, parser target, Go packages, configuration keys,
and CLI diagnostic behavior. A change to one of these contracts requires an
explicit compatibility review and an update to this page, the tests, and the
release checklist.

## Go toolchain and build targets

- **Minimum Go version:** Go 1.23, matching the `go` directive in `go.mod`.
  There is no declared maximum Go version. The release build uses the Go 1.23
  toolchain line or a later compatible toolchain.
- **Operating systems:** Linux, macOS, and Windows.
- **Architectures:** `amd64` and `arm64` on each supported operating system.
- **Validation boundary:** CI runs the full tests and vet on Linux/amd64. CI
  builds all packages for each of the six OS/architecture combinations. A
  successful cross-build does not claim native runtime validation for every
  target.

The Go module path remains `github.com/ayanozturk/go-php-parser` for Go imports
and existing PHP Strom integrations. The user-facing executable is `tusk` (or
`tusk.exe` on Windows); automatic config discovery uses `tusk.yaml`.

## PHP syntax scope

Tusk's grammar baseline and `syntax.ParseResult.PHPVersion` are PHP 8.3. The
lexer and parser also accept selected newer constructs, including PHP 8.4
property hooks. This is a parser target, not a promise that every PHP 8.3 or
8.4 grammar production is fully lowered or understood by every analysis rule.
The concrete syntax tree preserves source text, including syntax the current
AST lowering and analyzer do not understand. Malformed and unsupported input
may produce parser diagnostics or opaque/error nodes while preserving source.

The analyzer's support is narrower and rule-specific. See the [rule guide](rules/README.md)
and [analyzer capability matrix](analyser-capability-matrix.md) for covered
cases and known gaps. Tusk does not promise complete PHPStan compatibility.

## Go API boundary

The supported library API consists of exported identifiers in these packages:

- `analyse`, `ast`, `diag`, `lexer`, `overrides`, `printer`, `sharedcache`,
  `style`, `syntax`, and `token`.
- `config` is supported for loading and inspecting Tusk configuration.

The `cmd/...`, `command`, `helper`, `phpstubs`, `syntax/lower`, and `utils` packages are
implementation or developer-tool packages; their exported identifiers are not
covered by the 1.0 compatibility promise. The module path above is stable for
1.x. During 1.x, changes to the supported API preserve source compatibility;
breaking API or module-path changes require a new major version. Additions and
bug fixes remain possible. Diagnostic behavior changes are governed by the
diagnostic contract below.

## Configuration schema

Tusk searches only the current working directory for `tusk.yaml`. It does not
search parent directories. `-config <file>` before the command selects an
explicit config path. `tusk init` writes the following starting configuration:

```yaml
path: .
includes: []
extensions:
  - php
ignore:
  - vendor
  - node_modules
  - cache
  - .git
rules:
  - all
analysis_level: 0
```

The supported keys are:

| Key | YAML type | Meaning |
| --- | --- | --- |
| `path` | string | Root walked for project files. |
| `includes` | list of strings | Extra files/directories indexed for symbol resolution; their diagnostics are not reported. Composer `vendor` under `path` is indexed automatically. |
| `extensions` | list of strings | File extensions without a leading dot. Defaults from `init` to `php`. |
| `ignore` | list of strings | Directory names skipped while walking `path` and include directories. |
| `rules` | list of strings | Style rule codes to run; `all` selects every registered style rule. |
| `analysis_level` | integer or `null` | Enables cumulative analysis levels 0–10. An omitted or null value enables all registered levelled rules and unlevelled rules; a nonnegative explicit level runs levelled rules through that level. |
| `overrides` | mapping | Per-diagnostic class-name filters, described below. |

For example:

```yaml
path: src
includes:
  - vendor/acme/contracts
extensions: [php]
ignore: [cache, .git]
rules: [all]
analysis_level: 3
overrides:
  A.SYMBOL.UNKNOWN_CLASS:
    classes:
      - 'Legacy\MissingClass'
      - '/^Generated\\/'
```

An override matches an exact class name case-insensitively, or a regular
expression written between `/` characters. The `classes` list filters matching
class subjects for that diagnostic code. Negative `analysis_level` values are
rejected. The YAML decoder currently ignores unknown keys; misspelled keys may
therefore have no effect. `tusk config` prints the effective values.

The schema is kept compatible during 1.x. New keys may be added, but existing
key meaning and accepted types will not change in a backward-incompatible way.

## Diagnostics and process exit status

`analyze` prints human-readable diagnostics to standard output by default. A
diagnostic includes its file, location, message, and stable rule code. Results
are deterministic and source ordered. There is no supported JSON diagnostic
output contract in 1.0.

For `analyze`, process status is:

| Status | Meaning |
| ---: | --- |
| `0` | Analysis completed with no reported analysis or syntax diagnostics. |
| `1` | Analysis completed and reported one or more analysis or parser diagnostics. |
| `2` | Invocation, config, target discovery, include discovery, or file-read failure prevented a complete analysis. |

Style findings and parser errors printed by `style`, `ast`, or `tokens` do not
currently set status `1`; automation that needs a diagnostic-sensitive status
must use `analyze`. Unknown commands and project/config preflight failures use
status `2`. Exit behavior is covered by CLI tests and is part of the 1.0
contract.

Diagnostic codes are stable identifiers within 1.x. Message wording and
formatting may improve while retaining the code and source location. A rule
behavior change that can add or remove findings must update its [rule
documentation](rules/README.md) and relevant regression/differential fixtures.
