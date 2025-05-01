package command

import (
	"flag"
	"fmt"
	"go-phpcs/analyzer"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/printer"
	"go-phpcs/style"
	stylepsr12 "go-phpcs/style/psr12"
	"os"
	"sync"
)

// Command represents a command that can be executed
type Command struct {
	Name        string
	Description string
	Execute     func([]ast.Node, string)
}

// Commands maps command names to their implementations
var Commands = map[string]Command{
	"ast": {
		Name:        "ast",
		Description: "Print the Abstract Syntax Tree",
		Execute: func(nodes []ast.Node, filename string) {
			printer.PrintAST(nodes, 0)
		},
	},
	"tokens": {
		Name:        "tokens",
		Description: "Print the tokens from the lexer",
		Execute: func(nodes []ast.Node, filename string) {
			// This is a placeholder - the actual implementation is in main.go
		},
	},
	"style": {
		Name:        "style",
		Description: "Check code style (e.g., function naming)",
		Execute: func(nodes []ast.Node, filename string) {
			// Parse os.Args for output flag (since main.go may not use flag package for subcommands)
			outputFile := ""
			for i, arg := range os.Args {
				if (arg == "--output" || arg == "-o") && i+1 < len(os.Args) {
					outputFile = os.Args[i+1]
				}
			}

			var allIssues []style.StyleIssue
			checker := &style.ClassNameChecker{}
			allIssues = append(allIssues, checker.CheckIssues(nodes, filename)...)
			allIssues = append(allIssues, stylepsr12.RunAllPSR12Checks(filename)...)

			if outputFile != "" {
				f, err := os.Create(outputFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Could not create output file %s: %v\n", outputFile, err)
					return
				}
				defer f.Close()
				style.PrintPHPCSStyleOutputToWriter(f, allIssues)
				fmt.Fprintf(os.Stderr, "PHPCS-style report written to %s\n", outputFile)
			} else {
				style.PrintPHPCSStyleOutput(allIssues)
			}
		},
	},
	"analyse": {
		Name:        "analyse",
		Description: "Static analysis: unknown function calls (PoC)",
		Execute: func(nodes []ast.Node, filename string) {
			analyzer.AnalyzeUnknownFunctionCalls(nodes)
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

const ErrorLineFormat = "\t%s\n"

type ParseErrorDetail struct {
	File   string
	Errors []string
}

type MemStats struct {
	Start, End interface{}
}

func ProcessSingleFile(filePath, commandName string, debug bool) (int, int) {
	errList, lines := ProcessFileWithErrors(filePath, commandName, debug)
	totalParseErrors := 0
	if len(errList) > 0 {
		totalParseErrors += len(errList)
		if debug {
			fmt.Printf("\nParsing errors in %s (%d error(s)):\n", filePath, len(errList))
			for _, err := range errList {
				fmt.Printf(ErrorLineFormat, err)
			}
		}
	}
	return totalParseErrors, lines
}

func CollectLines(linesCh <-chan int, totalLines *int, done chan<- struct{}) {
	for lines := range linesCh {
		*totalLines += lines
	}
	done <- struct{}{}
}

func CollectErrors(errDetailCh <-chan ParseErrorDetail, totalParseErrors *int, done chan<- struct{}) {
	for errDetail := range errDetailCh {
		if len(errDetail.Errors) > 0 {
			*totalParseErrors += len(errDetail.Errors)
			fmt.Printf("\nParsing errors in %s (%d error(s)):\n", errDetail.File, len(errDetail.Errors))
			for _, err := range errDetail.Errors {
				fmt.Printf(ErrorLineFormat, err)
			}
		}
	}
	done <- struct{}{}
}

func Worker(fileCh <-chan string, errDetailCh chan<- ParseErrorDetail, linesCh chan<- int, commandName string, debug bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for filePath := range fileCh {
		errList, lines := ProcessFileWithErrors(filePath, commandName, debug)
		errDetailCh <- ParseErrorDetail{File: filePath, Errors: errList}
		linesCh <- lines
	}
}

func CountLines(input []byte) int {
	lineCount := 0
	for _, b := range input {
		if b == '\n' {
			lineCount++
		}
	}
	if len(input) > 0 && input[len(input)-1] != '\n' {
		lineCount++
	}
	return lineCount
}

func ExecuteCommand(commandName string, nodes []ast.Node, input []byte, filename string) {
	if cmd, exists := Commands[commandName]; exists {
		if commandName == "tokens" {
			l := lexer.New(string(input))
			for {
				tok := l.NextToken()
				if tok.Type == "T_EOF" {
					break
				}
				fmt.Printf("%s: %s @ %d:%d\n", tok.Type, tok.Literal, tok.Pos.Line, tok.Pos.Column)
			}
		} else {
			cmd.Execute(nodes, filename)
		}
	} else {
		fmt.Printf("Unknown command: %s\n", commandName)
		PrintUsage()
	}
}
