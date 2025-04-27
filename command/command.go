package command

import (
	"fmt"
	"go-phpcs/analyzer"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"go-phpcs/printer"
	"go-phpcs/style"
	"os"
	"sync"
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
	"analyse": {
		Name:        "analyse",
		Description: "Static analysis: unknown function calls (PoC)",
		Execute: func(nodes []ast.Node) {
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

func ProcessMultipleFiles(files []string, commandName string, debug bool, parallelism int) (int, int) {
	totalParseErrors := 0
	totalLines := 0
	var wg sync.WaitGroup
	fileCh := make(chan string)
	linesCh := make(chan int)
	errDetailCh := make(chan ParseErrorDetail)
	linesDone := make(chan struct{})
	errorsDone := make(chan struct{})

	go CollectLines(linesCh, &totalLines, linesDone)
	go CollectErrors(errDetailCh, &totalParseErrors, errorsDone)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go Worker(fileCh, errDetailCh, linesCh, commandName, debug, &wg)
	}

	for _, filePath := range files {
		fileCh <- filePath
	}
	close(fileCh)
	wg.Wait()
	close(linesCh)
	close(errDetailCh)
	<-linesDone
	<-errorsDone

	return totalParseErrors, totalLines
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

func ExecuteCommand(commandName string, nodes []ast.Node, input []byte) {
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
			cmd.Execute(nodes)
		}
	} else {
		fmt.Printf("Unknown command: %s\n", commandName)
		PrintUsage()
	}
}

func ProcessFileWithErrors(filePath, commandName string, debug bool) ([]string, int) {
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return nil, 0
	}
	lineCount := CountLines(input)
	l := lexer.New(string(input))
	p := parser.New(l, debug)
	nodes := p.Parse()
	errList := p.Errors()
	if len(errList) > 0 {
		return errList, lineCount
	}
	ExecuteCommand(commandName, nodes, input)
	return nil, lineCount
}

func ProcessFile(filePath, commandName string, debug bool) int {
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return 0
	}
	lineCount := CountLines(input)
	l := lexer.New(string(input))
	p := parser.New(l, debug)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		errCount := len(p.Errors())
		fmt.Printf("Parsing errors in %s (%d error(s)):\n", filePath, errCount)
		for _, err := range p.Errors() {
			fmt.Printf(ErrorLineFormat, err)
		}
		return lineCount
	}
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
			cmd.Execute(nodes)
		}
	} else {
		fmt.Printf("Unknown command: %s\n", commandName)
		PrintUsage()
	}
	return lineCount
}
