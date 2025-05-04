package command

import (
	"fmt"
	"go-phpcs/analyzer"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"go-phpcs/printer"
	"go-phpcs/sharedcache"
	"go-phpcs/style"
	"io"
	"io/ioutil"
)

type Command struct {
	Name             string
	Description      string
	Execute          func([]ast.Node, string, io.Writer)
	ExecuteWithRules func([]ast.Node, string, io.Writer, []string)
}

// Commands maps command names to their implementations
var Commands = map[string]Command{
	"ast": {
		Name:        "ast",
		Description: "Print the Abstract Syntax Tree",
		Execute: func(nodes []ast.Node, filename string, w io.Writer) {
			printer.PrintAST(nodes, 0)
		},
	},
	"tokens": {
		Name:        "tokens",
		Description: "Print the tokens from the lexer",
		Execute: func(nodes []ast.Node, filename string, w io.Writer) {
			// This is a placeholder - the actual implementation is in main.go
		},
	},
	"style": {
		Name:        "style",
		Description: "Check code style (e.g., function naming)",
		// Refactored for concurrent streaming: checkers run in goroutines, issues are streamed to output
		ExecuteWithRules: func(nodes []ast.Node, filename string, w io.Writer, allowedRules []string) {
			var allIssues []style.StyleIssue
			// checker := &style.ClassNameChecker{}
			// allIssues = append(allIssues, checker.CheckIssues(nodes, filename)...)
			content, err := sharedcache.GetCachedFileContent(filename)
			if err != nil {
				allIssues = append(allIssues, style.StyleIssue{
					Filename: filename,
					Line:     0,
					Type:     style.Error,
					Message:  "[PSR12] Could not load file content: " + err.Error(),
					Code:     "PSR12.Files.FileOpenError",
				})
			} else {
				allIssues = append(allIssues, style.RunSelectedRules(filename, content, nodes, allowedRules)...) // Unified registry
			}

			// If w is an IssueCollector, append to it; else, print immediately
			if collector, ok := w.(*style.IssueCollector); ok {
				for _, iss := range allIssues {
					collector.Append(iss)
				}
			} else {
				for _, iss := range allIssues {
					style.PrintPHPCSStyleIssueToWriter(w, iss)
				}
			}
		},
		// Execute will be assigned after map initialization to avoid cycle
		Execute: nil,
	},
	"analyse": {
		Name:        "analyse",
		Description: "Static analysis: unknown function calls (PoC)",
		Execute: func(nodes []ast.Node, filename string, w io.Writer) {
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

// Assign Execute for style after map initialization to avoid cycle
func init() {
	styleCmd := Commands["style"]
	styleCmd.Execute = func(nodes []ast.Node, filename string, w io.Writer) {
		styleCmd.ExecuteWithRules(nodes, filename, w, nil)
	}
	Commands["style"] = styleCmd
}

type ParseErrorDetail struct {
	File   string
	Errors []string
}

type MemStats struct {
	Start, End interface{}
}

func ProcessSingleFileWithWriter(filePath, commandName string, debug bool, w io.Writer) (int, int) {
	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(w, "Could not read file %s: %v\n", filePath, err)
		return 1, 0
	}
	lex := lexer.New(string(input))
	p := parser.New(lex, false)
	nodes := p.Parse()
	ExecuteCommand(commandName, nodes, input, filePath, w)
	return 0, CountLines(input)
}

func ProcessMultipleFilesWithWriter(files []string, commandName string, debug bool, parallelism int, w io.Writer) (int, int) {
	totalParseErrors := 0
	totalLines := 0
	for _, file := range files {
		err, lines := ProcessSingleFileWithWriter(file, commandName, debug, w)
		totalParseErrors += err
		totalLines += lines
	}
	return totalParseErrors, totalLines
}

func CollectLines(linesCh <-chan int, totalLines *int, done chan<- struct{}) {
	for lines := range linesCh {
		*totalLines += lines
	}
	done <- struct{}{}
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

func ExecuteCommand(commandName string, nodes []ast.Node, input []byte, filename string, w io.Writer) {
	if cmd, exists := Commands[commandName]; exists {
		cmd.Execute(nodes, filename, w)
	}
}
