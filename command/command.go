package command

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/printer"
	"go-phpcs/style"
)

// Command represents a command that can be executed
type Command struct {
	Name        string
	Description string
	Execute     func([]ast.Node)
}

// Commands maps command names to their implementations
var Commands = map[string]Command{
	"ast": {
		Name:        "ast",
		Description: "Print the Abstract Syntax Tree",
		Execute: func(nodes []ast.Node) {
			printer.PrintAST(nodes, 0)
		},
	},
	"tokens": {
		Name:        "tokens",
		Description: "Print the tokens from the lexer",
		Execute: func(nodes []ast.Node) {
			// This is a placeholder - the actual implementation is in main.go
		},
	},
	"style": {
		Name:        "style",
		Description: "Check code style (e.g., function naming)",
		Execute: func(nodes []ast.Node) {
			checker := &style.ClassNameChecker{}
			checker.Check(nodes)
		},
	},
}

// PrintUsage prints the usage information for all available commands
func PrintUsage() {
	fmt.Println("Usage: go run main.go <command> <php-file>")
	fmt.Println("Commands:")
	for _, cmd := range Commands {
		fmt.Printf("  %-8s %s\n", cmd.Name, cmd.Description)
	}
}
