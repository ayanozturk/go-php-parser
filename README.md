# go-php-parser

A PHP parser, code-style checker, and project-aware static analyzer written in Go.

`go-php-parser` turns PHP source into a detailed Abstract Syntax Tree, checks it against a registered set of style rules (PSR-12 and friends), and runs a project-aware analyzer that resolves symbols, types, and control flow across the configured files. Diagnostics are emitted in deterministic source order with stable exit codes, and the same engine backs the `analyze` command and the [PHP Strom](docs/full-static-analyser-target.md) language server.

The long-term target is a production-grade, full PHP static analyzer with cold full-project performance comparable to [Mago](https://github.com/carthage-software/mago), without trading semantic coverage or diagnostic quality for speed. See [Full Static Analyzer and Mago-Class Performance Target](docs/full-static-analyser-target.md) for the current pin, M1 status, ranked next actions, benchmark contract, and acceptance gates. Remaining CLI adoption work is in [Near-term CLI and adoption plan](docs/next-feature-plan.md) and is not the main stream.

## Features

### Language Support

- PHP 8+ syntax
- Function declarations with parameters
- Variable declarations and assignments
- Control structures (if, elseif, else)
- String literals (single and double quoted)
- String interpolation
- Integer and float literals
- Boolean literals (true, false)
- Null literal
- Comments (single-line and doc comments)
- Basic expressions and operators

### AST Features

- Detailed position tracking (line, column, offset)
- Hierarchical node structure
- Support for:
  - Function nodes
  - Variable nodes
  - Parameter nodes
  - Assignment nodes
  - Expression nodes
  - Control structure nodes
  - Comment nodes
  - Literal nodes (string, integer, float, boolean, null)

## Installation

```bash
git clone https://github.com/yourusername/go-php-parser.git
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

<!-- analysis-rule-level-table:start -->
| PHPStan level | Rules introduced | Cumulative levelled rules | Detail |
| ---: | ---: | ---: | --- |
| 0 | 1 | 1 | [Level 0 rules](docs/rules/level-0.md) |
| 1 | 1 | 2 | [Level 1 rules](docs/rules/level-1.md) |
| 2 | 6 | 8 | [Level 2 rules](docs/rules/level-2.md) |
| 3 | 5 | 13 | [Level 3 rules](docs/rules/level-3.md) |
| 4 | 1 | 14 | [Level 4 rules](docs/rules/level-4.md) |
| 5 | 0 | 14 | [Level 5 rules](docs/rules/level-5.md) |
| 6 | 0 | 14 | [Level 6 rules](docs/rules/level-6.md) |
| 7 | 1 | 15 | [Level 7 rules](docs/rules/level-7.md) |
| 8 | 1 | 16 | [Level 8 rules](docs/rules/level-8.md) |
| 9 | 0 | 16 | [Level 9 rules](docs/rules/level-9.md) |
| 10 | 2 | 18 | [Level 10 rules](docs/rules/level-10.md) |
| Unlevelled | 4 | 22 total registered | [Unlevelled rules](docs/rules/unlevelled.md) |
<!-- analysis-rule-level-table:end -->

Run `go run ./cmd/rule-inventory` after adding or moving an analysis rule. The Go test suite compares this table and each detail page's inventory metadata with the live registry, so a rule-count change cannot land without updating both. The linked level documents describe current coverage and known boundaries; update that prose in the same change as its rule or level.

### Listing All Style Rules

You can list all available style rule codes supported by this tool using the `list-style-rules` command. This is useful for discovering which rules you can enable or disable in your `config.yaml`.

Run the following command:

```bash
./go-phpcs list-style-rules
```

This will print a list of all registered style rule codes, for example:

```
Available style rule codes:
PSR12.Files.EndFileNoTrailingWhitespace
PSR12.Files.EndFileNewline
PSR12.Files.NoMultipleStatementsPerLine
PSR12.Files.NoSpaceBeforeSemicolon
PSR12.Files.NoBlankLineAfterPHPOpeningTag
PSR12.Classes.OpenBraceOnOwnLine
PSR12.Methods.VisibilityDeclared
PSR1.Classes.ClassDeclaration.PascalCase
PSR12.Classes.ClosingBraceOnOwnLine
...
```

You can then copy any of these codes into your `config.yaml` under the `rules:` section to customize which checks are performed.

## PSR-12 Style Checks

This parser implements several PSR-12 style checks, including:

- **No trailing whitespace** (`PSR12.Files.EndFileNoTrailingWhitespace`): Disallows trailing whitespace at the end of lines.
- **File must end with a single blank line** (`PSR12.Files.EndFileNewline`): Ensures files end with exactly one blank line.
- **No multiple statements per line** (`PSR12.Files.NoMultipleStatementsPerLine`): Disallows more than one statement (semicolon) per line.
- **No space before semicolon** (`PSR12.Files.NoSpaceBeforeSemicolon`): Disallows any space or tab before a semicolon at the end of a statement.
- **No blank line after opening <?php tag** (`PSR12.Files.NoBlankLineAfterPHPOpeningTag`): Disallows blank lines immediately after the opening PHP tag.
- **Class opening brace on its own line** (`PSR12.Classes.OpenBraceOnOwnLine`): Requires that the opening brace for a class, interface, trait, or enum must appear on its own line, with no leading or trailing whitespace.
- **Method visibility must be declared** (`PSR12.Methods.VisibilityDeclared`): Requires that every class method explicitly declares its visibility (`public`, `protected`, or `private`).

Style issues are reported per file and line, and can be extended by adding new checkers in the `style/` package.


## Available Style Rules

You can enable or disable specific code style rules using the `rules:` key in your `config.yaml`. If no rules are specified, all available rules are run.

**List of Available Rules:**

- `PSR12.Files.EndFileNoTrailingWhitespace`
- `PSR12.Files.EndFileNewline`
- `PSR12.Files.NoMultipleStatementsPerLine`
- `PSR12.Files.NoSpaceBeforeSemicolon`
- `PSR12.Files.NoBlankLineAfterPHPOpeningTag`
- `PSR12.Classes.OpenBraceOnOwnLine`
- `PSR12.Methods.VisibilityDeclared`
- `PSR12.Classes.ClosingBraceOnOwnLine`

| Rule Code                                 | Description                                |
|-------------------------------------------|--------------------------------------------|
| PSR12.Files.EndFileNoTrailingWhitespace   | Enforces no trailing whitespace on lines    |
| PSR12.Files.EndFileNewline                | File must end with a single blank line      |
| PSR12.Files.NoMultipleStatementsPerLine   | Disallows more than one statement (semicolon) per line |
| PSR12.Files.NoSpaceBeforeSemicolon        | Disallows any space or tab before a semicolon at the end of a statement |
| PSR12.Files.NoBlankLineAfterPHPOpeningTag | Disallows blank lines after the opening <?php tag |
| PSR1.Classes.ClassDeclaration.PascalCase | Enforces PascalCase for class names        |
| PSR12.Classes.ClosingBraceOnOwnLine         | Closing brace must be on its own line, and not followed by code or comments. Reports a syntax error if the file contains only a closing brace |

**Example config.yaml:**

```yaml
path: ./src
extensions:
  - php
ignore:
   - vendor
rules:
  - PSR12.Files.EndFileNoTrailingWhitespace
  - PSR12.Files.EndFileNewline
  - PSR12.Files.NoMultipleStatementsPerLine
  - PSR12.Files.NoSpaceBeforeSemicolon
  - PSR12.Files.NoBlankLineAfterPHPOpeningTag
  - PSR1.Classes.ClassDeclaration.PascalCase
  - PSR12.Classes.ClosingBraceOnOwnLine
```

Add or remove rule codes under `rules:` to control which checks are performed. If you don't specify `rules` it will execute all rules available.

### Basic Usage

```bash
go run main.go demo_project
```

This will parse the PHP files under the target directory and output the AST in a tree-like structure. You can also point it at a single file:

```bash
go run main.go demo_constants.php
```

### Directory Scanning & Parallelism

You can scan all PHP files in a directory as defined in `config.yaml`:

```bash
go run main.go
```

To control parallelism (number of concurrent workers), use the `-p` flag. By default, the number of workers is set to the number of CPU cores on your machine:

```bash
go run main.go -p 4   # Use 4 workers in parallel
```

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

### Full-Analyser Benchmark

To measure the analysis engine itself (not the style checker) against the checked-in `test_projects` corpus — index-only, process-cold full analysis, and warm-loop full analysis, with timing, RSS, and diagnostic counts per the [full-static-analyser benchmark contract](docs/full-static-analyser-target.md#comparable-performance-contract):

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

`test_projects/*` (other than `manifest.json`) are fetched on demand, not committed — each is large (tens to hundreds of MB) and Git has no reliable way to pin an external directory's exact revision without either committing its full content or a real submodule. `go run ./cmd/fetch-test-projects` reads `test_projects/manifest.json` and checks out each project's exact pinned commit (a shallow, single-commit fetch, not a full clone) into `test_projects/<name>`, skipping projects already at the pinned commit. The manifest records the Mago benchmark's three required workloads (`php-standard-library`, `wordpress-develop`, `magento2`) alongside this project's own representative framework corpora (Composer, Drupal, Laravel, PHPUnit, Symfony), each with its exact commit per the [comparable-performance contract](docs/full-static-analyser-target.md#comparable-performance-contract).

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
    "go-php-parser/lexer"
    "go-php-parser/parser"
    "go-php-parser/ast"
)

func main() {
    // Read PHP file
    input := `<?php
    function test($param) {
        echo "Hello, $param!";
    }`

    // Create lexer
    l := lexer.New(input)

    // Create parser
    p := parser.New(l)

    // Parse the input
    nodes := p.Parse()

    // Check for errors
    if len(p.Errors()) > 0 {
        fmt.Println("Parsing errors:")
        for _, err := range p.Errors() {
            fmt.Printf("\t%s\n", err)
        }
        return
    }

    // Print AST
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
