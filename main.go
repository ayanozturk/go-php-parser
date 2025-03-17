package main

import (
	"fmt"
	"go-phpcs/command"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		command.PrintUsage()
		os.Exit(1)
	}

	commandName := os.Args[1]
	filePath := os.Args[2]

	// Read the PHP file
	input, err := os.ReadFile(filePath)
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

	// Handle commands
	if cmd, exists := command.Commands[commandName]; exists {
		cmd.Execute(nodes)
	} else {
		fmt.Printf("Unknown command: %s\n", commandName)
		command.PrintUsage()
		os.Exit(1)
	}
}
