# Go PHP Parser

A PHP parser and code style checker written in Go that generates an Abstract Syntax Tree (AST) from PHP source code and applies style rules to generate a report.

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

To use the style checker against your codebase, first build a the project

```bash
make build
```

This will generate a binary named `go-phpcs`

- Copy this binary, together with `config.yaml` file in this repository into your project.
- Modify `config.yaml` file to target the directory you need PHP style checks.
- Run the style checker

```bash
./go-phpcs
```

Optionally export the report into a file

```bash
./go-phpcs -o report.log
```

### Option 2

Clone your project into a folder within this project.

Update `config.yaml` with your folder name.

Run the style checks:

```bash
make run
```

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

Style issues are reported per file and line, and can be extended by adding new checkers in the `style/psr12` package.


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
go run main.go examples/test.php
```

This will parse the PHP file and output the AST in a tree-like structure.

### Directory Scanning & Parallelism

You can scan all PHP files in a directory as defined in `config.yaml`:

```bash
go run main.go
```

To control parallelism (number of concurrent workers), use the `-p` flag. By default, the number of workers is set to the number of CPU cores on your machine:

```bash
go run main.go -p 4   # Use 4 workers in parallel
```

### Performance Output

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