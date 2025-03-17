package main

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <php-file>")
		os.Exit(1)
	}

	// Read the PHP file
	input, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	// Create new lexer
	l := lexer.New(string(input))

	// Create new parser
	p := parser.New(l)

	// Parse the input
	nodes := p.Parse()

	// Check for parsing errors
	if len(p.Errors()) > 0 {
		fmt.Println("Parsing errors:")
		for _, err := range p.Errors() {
			fmt.Printf("\t%s\n", err)
		}
		os.Exit(1)
	}

	// Print the AST
	fmt.Println("Abstract Syntax Tree:")
	ast.PrintAST(nodes, 0)
}
