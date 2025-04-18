package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Path       string   `yaml:"path"`
	Extensions []string `yaml:"extensions"`
	Ignore     []string `yaml:"ignore"`
}

func main() {
	totalParseErrors := 0
	// Load the configuration
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	filesToScan, err := getFilesToScan(config)
	if err != nil {
		log.Fatalf("Error scanning files: %v", err)
	}

	// Print the files to scan
	// fmt.Println("Files to scan:")
	// for _, file := range filesToScan {
	// 	fmt.Println(file)
	// }

	debug := flag.Bool("debug", false, "Enable debug mode to show parsing errors")
	parallelism := flag.Int("p", runtime.NumCPU(), "Number of files to process in parallel")
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

	start := time.Now()
	totalLines := 0

	if len(flag.Args()) > 1 {
		filePath := flag.Args()[1]
		if filePath == "" {
			fmt.Println("No file specified for parsing.")
			command.PrintUsage()
			os.Exit(1)
		}
		errCount, lines := processFileWithErrors(filePath, commandName, *debug)
		totalParseErrors += errCount
		totalLines += lines
	} else {
		if len(filesToScan) == 0 {
			fmt.Println("No files to scan.")
			os.Exit(1)
		}

		files := filesToScan
		var wg sync.WaitGroup
		fileCh := make(chan string)
		linesCh := make(chan int)

		// Collector goroutine
		errCh := make(chan int)
		go func() {
			for lines := range linesCh {
				totalLines += lines
			}
		}()
		go func() {
			for errs := range errCh {
				totalParseErrors += errs
			}
		}()

		// Start worker goroutines
		workerCount := *parallelism
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for filePath := range fileCh {
					errCount, lines := processFileWithErrors(filePath, commandName, *debug)
					errCh <- errCount
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
		close(errCh)
	}

	elapsed := time.Since(start).Seconds()
	fmt.Printf("\nScan completed in %.2f seconds\n", elapsed)
	if elapsed > 0 {
		fmt.Printf("Total lines scanned: %d\n", totalLines)
		fmt.Printf("Lines per second: %.2f\n", float64(totalLines)/elapsed)
	} else {
		fmt.Printf("Total lines scanned: %d\n", totalLines)
		fmt.Printf("Lines per second: N/A (too fast to measure)\n")
	}
	// End stats print block
	fmt.Printf("Total parsing errors: %d\n", totalParseErrors)
}

// processFileWithErrors returns (errorCount, lineCount)
func processFileWithErrors(filePath, commandName string, debug bool) (int, int) {
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return 0, 0
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
	errCount := len(p.Errors())
	if errCount > 0 {
		fmt.Printf("Parsing errors in %s (%d error(s)):\n", filePath, errCount)
		for _, err := range p.Errors() {
			fmt.Printf("\t%s\n", err)
		}
		return errCount, lineCount
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
	return 0, lineCount
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

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func getFilesToScan(config *Config) ([]string, error) {
	var filesToScan []string

	err := filepath.Walk(config.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		for _, ignore := range config.Ignore {
			if info.IsDir() && info.Name() == ignore {
				return filepath.SkipDir
			}
		}

		// Check file extensions
		if !info.IsDir() {
			for _, ext := range config.Extensions {
				if filepath.Ext(path) == "."+ext {
					filesToScan = append(filesToScan, path)
					break
				}
			}
		}

		return nil
	})

	return filesToScan, err
}
