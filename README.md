# go-php-parser

A PHP parser, code-style checker, and project-aware static analyzer written in Go.

`go-php-parser` parses PHP into a lossless concrete syntax tree and a lower-level AST, applies configurable style rules, and analyzes symbols, PHPDoc types, and control flow across a project. It provides the analysis engine used by the `analyze` command and the PHP Strom language server, with diagnostics emitted in deterministic source order.

The long-term target is a production-grade, full PHP static analyzer. The active implementation checklist is in [plan.MD](plan.MD); future work and sequencing are maintained in the [project roadmap](roadmap.md).

## Features

- **PHP parsing:** PHP 8 syntax, a lossless concrete syntax tree that preserves source text and trivia, an AST lowering API, and source positions for syntax nodes.
- **Editor support:** tolerant parsing with error nodes, reusable parse results, and the shared analysis engine used by the [PHP Strom language server](https://github.com/ayanozturk/vscode-php-strom).
- **Project analysis:** cross-file symbol resolution, PHPDoc types and aliases, generic and shape types, argument/return/property checks, and control-flow narrowing and joins.
- **Configurable diagnostics:** analysis levels 0–10 plus unlevelled rules, deterministic diagnostic ordering, and stable command exit codes.
- **Code style:** configurable style rules, including the PSR-12 checks documented below.
- **Developer tooling:** command-line project discovery, configuration inspection, file listing, style-rule inventory, corpus checks, and benchmarks.

The analyzer is under active development; see [plan.MD](plan.MD) for the current implementation scope and known acceptance gates. Release scope and the open 1.0 readiness checks are tracked in [docs/releasing-1.0.md](docs/releasing-1.0.md).

## Installation

```bash
git clone https://github.com/ayanozturk/go-php-parser.git
cd go-php-parser
go mod download
```

## Usage

### Option 1

Build the binary once:

```bash
make build
```

This produces a binary named `go-phpcs`.

#### Pointing go-phpcs at your project

The binary auto-discovers a config in the current working directory, in this order:

1. `tusk.yaml`
2. `go-phpcs.yaml`
3. `go-phpcs.yml`
4. `config.yaml`

To bootstrap a fresh project, generate a default `config.yaml` and edit it:

```bash
./go-phpcs init
```

You can also place the binary alongside an existing `config.yaml` from this repo and edit it to target the directory you want to check. The binary will pick up the nearest config automatically.

To inspect which config, path, extensions, ignore list, rules, and analysis level are actually in effect, run:

```bash
./go-phpcs config
```

To print exactly which files the resolved config will scan:

```bash
./go-phpcs list-files
```

#### Running the style checker

```bash
./go-phpcs
```

Optionally export the report into a file:

```bash
./go-phpcs -o report.log
```

### Option 2

Clone your project into a folder within this project (for example `demo_project/`).

Update `config.yaml` with your folder name.

Run the style checks:

```bash
make run
```

### Static analysis

Run project-aware static analysis for the files selected by `config.yaml`:

```bash
./go-phpcs analyze
```

Or analyze one file using the same project pipeline:

```bash
./go-phpcs analyze src/Example.php
```

Or scope the analysis to a subfolder (the whole project is still indexed for cross-file symbol resolution, but only files under `src/Module` are reported on):

```bash
./go-phpcs analyze src/Module
```

The folder walk respects the configured `extensions` and `ignore` lists, so `./go-phpcs analyze src` and a config pointed at `src` produce the same file set.

Set `analysis_level` to zero or greater to run rules up to that level. Leaving it unset runs every registered analysis rule.

```yaml
path: ./src
extensions:
  - php
analysis_level: 0
```

The analyzer parses each selected file once, builds one immutable project snapshot, and emits diagnostics in deterministic source order. Exit code `0` means clean, `1` means analysis or parser findings, and `2` means an invocation, configuration, discovery, or file-read failure.

#### Analysis rules by PHPStan level

The counts below are registered engine rules, not PHPStan error-identifier counts. A single engine rule can cover several PHPStan identifiers through one shared traversal. “Cumulative” reflects the rules enabled when `analysis_level` is set to that level. Unlevelled rules run only when `analysis_level` is omitted.

For a rule-by-rule guide with examples and reasons for each check, start at the [rule documentation index](docs/rules/README.md). It links to the analysis rules by level and to the style rule guide.

<!-- analysis-rule-level-table:start -->
| PHPStan level | Rules introduced | Cumulative levelled rules | Detail |
| ---: | ---: | ---: | --- |
| 0 | 2 | 2 | [Level 0 rules](docs/rules/level-0.md) |
| 1 | 2 | 4 | [Level 1 rules](docs/rules/level-1.md) |
| 2 | 17 | 21 | [Level 2 rules](docs/rules/level-2.md) |
| 3 | 5 | 26 | [Level 3 rules](docs/rules/level-3.md) |
| 4 | 1 | 27 | [Level 4 rules](docs/rules/level-4.md) |
| 5 | 1 | 28 | [Level 5 rules](docs/rules/level-5.md) |
| 6 | 5 | 33 | [Level 6 rules](docs/rules/level-6.md) |
| 7 | 1 | 34 | [Level 7 rules](docs/rules/level-7.md) |
| 8 | 1 | 35 | [Level 8 rules](docs/rules/level-8.md) |
| 9 | 1 | 36 | [Level 9 rules](docs/rules/level-9.md) |
| 10 | 0 | 36 | [Level 10 rules](docs/rules/level-10.md) |
| Unlevelled | 4 | 40 total registered | [Unlevelled rules](docs/rules/unlevelled.md) |
<!-- analysis-rule-level-table:end -->

Run `go run ./cmd/rule-inventory` after adding or moving an analysis rule. The Go test suite compares this table and each detail page's inventory metadata with the live registry. It also checks that every registered analysis and style rule has a documented entry, example, and rationale. Update the entry and coverage notes in the same change as any rule addition, removal, rename, or behavior change; see the rule-documentation instructions in [AGENTS.md](AGENTS.md).

### Listing All Style Rules

List the registered style rule codes supported by this tool:

```bash
./go-phpcs list-style-rules
```

The [style rule guide](docs/rules/style.md) explains every registered style check, with an example and why it helps. You can enable or disable codes under `rules:` in `config.yaml`; if the setting is omitted, all registered style rules run.

### Inspecting syntax

Print the lowered AST for one PHP file:

```bash
./go-phpcs ast path/to/Example.php
```

To check all files selected by the project configuration, use `./go-phpcs` (style checks) or `./go-phpcs analyze` (project-aware analysis). Use `./go-phpcs -p 4 analyze` to set the worker count; by default it uses the number of CPUs visible to Go.

### Compatibility Metrics

First, fetch the pinned corpora (not committed to this repository — see [test_projects/manifest.json](test_projects/manifest.json)):

```bash
go run ./cmd/fetch-test-projects
```

To track parser compatibility progress across the checked-in corpus under `test_projects`, run:

```bash
make compat-metrics
```

This prints overall file compatibility, per-project compatibility, total parse errors, and a small sample of the first failing files per project.

You can also emit a machine-readable snapshot for tracking over time:

```bash
go run ./cmd/compat-metrics -json -output compatibility-report.json
```

Useful flags:

- `-root` to scan a different corpus root
- `-workers` to control parallelism
- `-top` to control how many failing-file examples are shown per project

Parser compatibility and PHPStan diagnostic compatibility are separate metrics. To report a corpus-scoped `Level N: X% PHPStan compatible`, use the exact-location precision/recall/F1 report documented in [docs/phpstan-compatibility-metric.md](docs/phpstan-compatibility-metric.md):

```bash
go run ./cmd/phpstan-compat \
  --root /path/to/project \
  --paths src,tests \
  --index-paths vendor \
  --phpstan-bin /path/to/project/vendor/bin/phpstan \
  --phpstan-config /path/to/project/phpstan.neon
```

### Full-Analyser Benchmark

Performance is tracked against our own best accepted results. For each change, use the same pinned workload and environment, confirm complete file and diagnostic parity, and keep timing and peak memory at or better than the current best. The [1.0 release checklist](docs/releasing-1.0.md) defines the measurement gate.

To measure the analysis engine itself (not the style checker) against the checked-in `test_projects` corpus — index-only, process-cold full analysis, and warm-loop full analysis, with timing, RSS, and diagnostic counts per the [rolling performance process](roadmap.md#6-improve-and-maintain-full-analysis-performance):

```bash
go run ./cmd/benchmark --root test_projects/symfony --json --output benchmark-report.json
```

Or a human-readable summary:

```bash
go run ./cmd/benchmark --root test_projects/phpunit
```

For a selected-path workload, pass the same source/include boundary used by the reference analyser. Paths are relative to `--root`; missing paths fail instead of silently shrinking the corpus:

```bash
go run ./cmd/benchmark \
  --root test_projects/wordpress-develop \
  --paths src,tests,vendor \
  --excludes src/js \
  --json
```

Cold-full-analysis runs each re-exec the binary as a fresh subprocess (10 by default) so no in-process cache state leaks between measured runs. The parent times the entire child lifetime, including startup, discovery, reads, parsing, indexing, analysis, reduction, and result serialization. Warm-full-analysis loops the indexed analysis pipeline in a single process after one unmeasured warmup iteration. Incremental-edit timing is reported as unsupported — the engine has no incremental invalidation API yet.

### Pinned Benchmark Corpora

`test_projects/*` (other than `manifest.json`) are fetched on demand, not committed — each is large (tens to hundreds of MB) and Git has no reliable way to pin an external directory's exact revision without either committing its full content or a real submodule. `go run ./cmd/fetch-test-projects` reads `test_projects/manifest.json` and checks out each project's exact pinned commit (a shallow, single-commit fetch, not a full clone) into `test_projects/<name>`, skipping projects already at the pinned commit. The manifest records the parser's required workloads (`php-standard-library`, `wordpress-develop`, `magento2`) alongside representative framework corpora (Composer, Drupal, Laravel, PHPUnit, Symfony), each with its exact commit per the [rolling performance process](roadmap.md#6-improve-and-maintain-full-analysis-performance).

```bash
go run ./cmd/fetch-test-projects                       # fetch everything in the manifest
go run ./cmd/fetch-test-projects --only psl,magento2    # fetch a subset
go run ./cmd/fetch-test-projects --force                # re-fetch even if already at the pinned commit
```

To re-pin a project to a newer revision, update its `commit` (and `ref`, for readability) in `test_projects/manifest.json` and re-run with `--force`.

Useful flags:

- `--root` corpus root to scan
- `--paths` comma-separated paths within the root to scan
- `--excludes` comma-separated paths within the root to exclude
- `--level` analysis rule level filter (`-1` = run every registered rule)
- `--cold-runs` number of measured process-cold runs (contract minimum is 10)
- `--warm-iterations` in-process warm-loop iterations, including the unmeasured warmup
- `--skip-cold` skip the process-cold subprocess runs for a quick check
- `--cpuprofile`/`--memprofile` write a `go tool pprof`-compatible CPU or heap profile from a single in-process full-analysis run (bypasses the cold/warm harness so the profiler attaches directly to the profiled work); pair with `--profile-iterations` to profile several in-process passes at once

For a faster syntax-optimization feedback loop, run `make syntax-bench-quick`.
It preloads a deterministic 128-file WordPress sample, includes the largest-file
tail, verifies lossless identity before timing, and reports CST parsing, AST
lowering, and combined allocations separately. Override the sample with
`SYNTAX_BENCH_ROOT=/path`, `SYNTAX_BENCH_FILES=256`, or
`SYNTAX_BENCH_COUNT=10`. Use this directional benchmark to reject weak ideas;
the full corpus identity, accounting, and cold-CV protocol remains the gate for
accepted performance claims.



After scanning, the tool will print performance statistics:

```
Scan completed in 1.55 seconds
Total lines scanned: 1653877
Lines per second: 1063784.86
Total parsing errors: 0
HeapAlloc: 148.56 MB
Sys: 298.92 MB
```

### Configuration

File scanning is controlled by `config.yaml`:

```yaml
path: ./demo_project
extensions:
  - php
ignore:
  # - vendor
```
- `path`: Directory to scan
- `extensions`: File extensions to include
- `ignore`: Directories to skip (uncomment to enable)

### Programmatic Usage

```go
package main

import (
    "fmt"

    "go-php-parser/ast"
    "go-php-parser/syntax"
)

func main() {
    src := []byte(`<?php
    function test($param) {
        echo "Hello, $param!";
    }`)

    nodes, diags := syntax.ParseAST(src)

    if len(diags) > 0 {
        fmt.Println("Parsing errors:")
        for _, d := range diags {
            fmt.Printf("\t%s\n", d.Message)
        }
        return
    }

    ast.PrintAST(nodes, 0)
}
```

## Project Structure

```
go-php-parser/
├── ast/         # AST node definitions
├── lexer/       # Tokenizer implementation
├── parser/      # Parser implementation
├── token/       # Token type definitions
├── examples/    # Example PHP files
└── main.go      # Main entry point
```

## AST Node Types

### Core Nodes

- `Node` - Base interface for all AST nodes
- `Position` - Line/column/offset information

### Expression Nodes

- `Identifier` - Variable or function names
- `VariableNode` - PHP variables ($var)
- `StringLiteral` - String literals
- `InterpolatedStringLiteral` - Strings with variable interpolation
- `IntegerLiteral` - Integer literals
- `FloatLiteral` - Floating-point literals
- `BooleanLiteral` - Boolean literals (true/false)
- `NullLiteral` - Null literal
- `BinaryExpr` - Binary expressions
- `FunctionCall` - Function calls

### Statement Nodes

- `FunctionNode` - Function declarations
- `ParameterNode` - Function parameters
- `AssignmentNode` - Variable assignments
- `ExpressionStmt` - Expression statements
- `ReturnNode` - Return statements
- `IfNode` - If statements
- `ElseIfNode` - Elseif clauses
- `ElseNode` - Else clauses
- `WhileNode` - While loops
- `CommentNode` - Comments

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
