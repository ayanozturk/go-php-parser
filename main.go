package main

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"os"
)

func main() {
	// Example PHP code
	input := `<?php
function hello() {
	$message = "Hello World";
}
`

	// Create new lexer
	l := lexer.New(input)

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
