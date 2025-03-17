package main

import (
	"fmt"
	"os"

	"go-phpcs/parser"
)

func main() {
	// PHP code to be parsed
	code := `<?php

function test($foo)
{
    var_dump($foo);
}
`

	// Create a parser instance for the newest supported PHP version
	parserFactory := parser.NewParserFactory()
	phpParser := parserFactory.CreateForNewestSupportedVersion()

	// Parse the code into an AST
	ast, err := phpParser.Parse(code)
	if err != nil {
		fmt.Printf("Parse error: %s\n", err.Error())
		os.Exit(1)
	}

	// Create a node dumper and print the AST in human-readable form
	dumper := parser.NewNodeDumper()
	fmt.Println(dumper.Dump(ast))

	// Example of traversing and modifying the AST
	traverser := parser.NewNodeTraverser()
	traverser.AddVisitor(&FunctionBodyCleaner{})

	// Apply the traversal to modify the AST
	ast = traverser.Traverse(ast)
	fmt.Println("\nAfter modification:")
	fmt.Println(dumper.Dump(ast))

	// Convert the modified AST back to PHP code
	prettyPrinter := parser.NewStandardPrettyPrinter()
	fmt.Println("\nRegenerated PHP code:")
	fmt.Println(prettyPrinter.PrettyPrintFile(ast))
}

// FunctionBodyCleaner is a visitor that removes function bodies
type FunctionBodyCleaner struct {
	parser.NodeVisitorAbstract
}

// EnterNode is called for each node in the AST during traversal
func (v *FunctionBodyCleaner) EnterNode(node parser.Node) (parser.Node, error) {
	// Check if the node is a function declaration
	if function, ok := node.(*parser.StmtFunction); ok {
		// Clear the function body
		function.Stmts = []parser.Node{}
	}
	return node, nil
}
