package command

import (
	"fmt"
	"go-phpcs/ast"
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
			fmt.Println("Abstract Syntax Tree:")
			ast.PrintAST(nodes, 0)
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
