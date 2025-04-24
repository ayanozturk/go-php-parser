package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/config"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

type ParseErrorDetail struct {
	File   string
	Errors []string
}

func main() {
	totalParseErrors := 0

	c, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	filesToScan, err := config.GetFilesToScan(c)
	if err != nil {
		log.Fatalf("Error scanning files: %v", err)
	}

	debug := flag.Bool("debug", false, "Enable debug mode to show parsing errors")
	parallelism := flag.Int("p", 2, "Number of files to process in parallel (default 2 for memory efficiency)")
	flag.Parse()

	if len(flag.Args()) < 2 && len(filesToScan) == 0 {
		fmt.Println("Usage: go-phpcs <command> <file>")
		command.PrintUsage()
		os.Exit(1)
	}

	commandName := "style"
	if len(flag.Args()) > 0 {
		commandName = flag.Args()[0]
	}

	fmt.Println("Command:", commandName)

	start := time.Now()
	totalLines := 0

	// Track memory usage
	var memStart, memEnd runtime.MemStats
	runtime.ReadMemStats(&memStart)

	if len(flag.Args()) > 1 {
		filePath := flag.Args()[1]
		if filePath == "" {
			fmt.Println("No file specified for parsing.")
			command.PrintUsage()
			os.Exit(1)
		}
		errList, lines := processFileWithErrors(filePath, commandName, *debug)
		if len(errList) > 0 {
			totalParseErrors += len(errList)
			if *debug {
				fmt.Printf("\nParsing errors in %s (%d error(s)):\n", filePath, len(errList))
				for _, err := range errList {
					fmt.Printf("\t%s\n", err)
				}
			}
		}
		totalLines += lines
	} else {
		if len(filesToScan) == 0 {
			fmt.Println("No files to scan.")
			os.Exit(1)
		}

		files := filesToScan
		if len(files) > 3500 {
			files = files[:3500]
		}
		var wg sync.WaitGroup
		fileCh := make(chan string)
		linesCh := make(chan int)
		errDetailCh := make(chan ParseErrorDetail)

		// Collector goroutine
		go func() {
			for lines := range linesCh {
				totalLines += lines
			}
		}()
		go func() {
			for errDetail := range errDetailCh {
				if len(errDetail.Errors) > 0 {
					totalParseErrors += len(errDetail.Errors)
					fmt.Printf("\nParsing errors in %s (%d error(s)):\n", errDetail.File, len(errDetail.Errors))
					for _, err := range errDetail.Errors {
						fmt.Printf("\t%s\n", err)
					}
				}
			}
		}()

		// Start worker goroutines
		workerCount := *parallelism
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for filePath := range fileCh {
					errList, lines := processFileWithErrors(filePath, commandName, *debug)
					// Always send a value to errDetailCh, even if errList is empty
					errDetailCh <- ParseErrorDetail{File: filePath, Errors: errList}
					linesCh <- lines
				}
			}()
		}

		// Send files to workers
		for _, filePath := range files {
			fileCh <- filePath
		}
		close(fileCh)
		wg.Wait()
		close(linesCh)
		close(errDetailCh)
	}

	runtime.GC()
	runtime.ReadMemStats(&memEnd)
	elapsed := time.Since(start).Seconds()
	fmt.Printf("\nScan completed in %.2f seconds\n", elapsed)
	if elapsed > 0 {
		fmt.Printf("Total lines scanned: %d\n", totalLines)
		fmt.Printf("Lines per second: %.2f\n", float64(totalLines)/elapsed)
	} else {
		fmt.Printf("Total lines scanned: %d\n", totalLines)
		fmt.Printf("Lines per second: N/A (too fast to measure)\n")
	}
	fmt.Printf("Total parsing errors: %d\n", totalParseErrors)
	fmt.Printf("HeapAlloc: %.2f MB\n", float64(memEnd.HeapAlloc)/(1024*1024))
	fmt.Printf("Sys: %.2f MB\n", float64(memEnd.Sys)/(1024*1024))
}

// processFileWithErrors returns (errorCount, lineCount)
// processFileWithErrors returns (errorList, lineCount)
func processFileWithErrors(filePath, commandName string, debug bool) ([]string, int) {
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return nil, 0
	}
	lineCount := 0
	for _, b := range input {
		if b == '\n' {
			lineCount++
		}
	}
	if len(input) > 0 && input[len(input)-1] != '\n' {
		lineCount++
	}
	l := lexer.New(string(input))
	p := parser.New(l, debug)
	nodes := p.Parse()
	errList := p.Errors()
	if len(errList) > 0 {
		// Do not print here; errors will be printed at the end
		return errList, lineCount
	}
	if cmd, exists := command.Commands[commandName]; exists {
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
		command.PrintUsage()
	}
	return nil, lineCount
}

// --- Helper function to process a file ---
func processFile(filePath, commandName string, debug bool) int {
	// Read the PHP file
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return 0
	}

	// Count lines in the file
	lineCount := 0
	for _, b := range input {
		if b == '\n' {
			lineCount++
		}
	}
	// If file is not empty and does not end with a newline, count the last line
	if len(input) > 0 && input[len(input)-1] != '\n' {
		lineCount++
	}

	// Create new lexer
	l := lexer.New(string(input))

	// Create new parser
	p := parser.New(l, debug)

	// Parse the input
	nodes := p.Parse()

	// Check for parsing errors
	if len(p.Errors()) > 0 {
		errCount := len(p.Errors())
		fmt.Printf("Parsing errors in %s (%d error(s)):\n", filePath, errCount)
		for _, err := range p.Errors() {
			fmt.Printf("\t%s\n", err)
		}
		return lineCount
	}

	// Handle commands
	if cmd, exists := command.Commands[commandName]; exists {
		if commandName == "tokens" {
			// For tokens command, create a new lexer with the original input
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
		command.PrintUsage()
	}
	return lineCount
}
