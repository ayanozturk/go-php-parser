package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"os"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode to show parsing errors")
	flag.Parse()

	if len(flag.Args()) < 2 {
		command.PrintUsage()
		os.Exit(1)
	}

	commandName := flag.Args()[0]
	filePath := flag.Args()[1]

	// Read the PHP file
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	// Create new lexer
	l := lexer.New(string(input))

	// Create new parser
	p := parser.New(l, *debug)

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
