# Go PHP Parser

A PHP parser written in Go that generates an Abstract Syntax Tree (AST) from PHP source code.

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
Scan completed in 0.92 seconds
Total lines scanned: 1653877
Lines per second: 1790022.29
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