package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/config"
	"go-phpcs/lexer"
	"go-phpcs/style"
	"go-phpcs/parser"
	"io"
	"log"
	"os"
	"runtime"
	"time"
)

const errorLineFormat = "\t%s\n"

type memStats struct {
	start, end runtime.MemStats
}

func printSummary(w io.Writer, totalParseErrors, totalLines int, elapsed float64, mem memStats) {
	fmt.Fprintf(w, "\nScan completed in %.2f seconds\n", elapsed)
	if elapsed > 0 {
		fmt.Fprintf(w, "Total lines scanned: %d\n", totalLines)
		fmt.Fprintf(w, "Lines per second: %.2f\n", float64(totalLines)/elapsed)
	} else {
		fmt.Fprintf(w, "Total lines scanned: %d\n", totalLines)
		fmt.Fprintf(w, "Lines per second: N/A (too fast to measure)\n")
	}
	fmt.Fprintf(w, "Total parsing errors: %d\n", totalParseErrors)
	fmt.Fprintf(w, "HeapAlloc: %.2f MB\n", float64(mem.end.HeapAlloc)/(1024*1024))
	fmt.Fprintf(w, "Sys: %.2f MB\n", float64(mem.end.Sys)/(1024*1024))
}

func trackMemoryUsage(mem *memStats, atStart bool) {
	if atStart {
		runtime.ReadMemStats(&mem.start)
	} else {
		runtime.GC()
		runtime.ReadMemStats(&mem.end)
	}
}

func main() {
	outputFile := flag.String("output", "", "Write all output (including summary) to this file")
	outputFileShort := flag.String("o", "", "Write all output (including summary) to this file (shorthand)")

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

	var outWriter io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Could not create output file %s: %v", *outputFile, err)
		}
		defer f.Close()
		outWriter = f
	} else if *outputFileShort != "" {
		f, err := os.Create(*outputFileShort)
		if err != nil {
			log.Fatalf("Could not create output file %s: %v", *outputFileShort, err)
		}
		defer f.Close()
		outWriter = f
	}

	if len(flag.Args()) < 2 && len(filesToScan) == 0 {
		fmt.Fprintln(outWriter, "Usage: go-phpcs <command> <file>")
		command.PrintUsage()
		os.Exit(1)
	}

	commandName := "style"
	if len(flag.Args()) > 0 {
		commandName = flag.Args()[0]
	}

	fmt.Fprintln(outWriter, "Command:", commandName)

	start := time.Now()
	var mem memStats
	trackMemoryUsage(&mem, true)

	totalParseErrors := 0
	totalLines := 0

	if len(flag.Args()) > 1 {
		filePath := flag.Args()[1]
		if filePath == "" {
			fmt.Fprintln(outWriter, "No file specified for parsing.")
			command.PrintUsage()
			os.Exit(1)
		}
		totalParseErrors, totalLines = command.ProcessFile(filePath, commandName, *debug, outWriter), 0
	} else {
		if len(filesToScan) == 0 {
			fmt.Fprintln(outWriter, "No files to scan.")
			os.Exit(1)
		}
		// If command is style, use ExecuteWithRules to allow rule filtering
		if commandName == "style" {
			var allIssues []style.StyleIssue
			for _, file := range filesToScan {
				input, err := os.ReadFile(file)
				if err != nil {
					fmt.Fprintf(outWriter, "Could not read file %s: %v\n", file, err)
					continue
				}
				lex := lexer.New(string(input))
				p := parser.New(lex, false)
				nodes := p.Parse()
				// Collect issues for this file
				var fileIssues []style.StyleIssue
				issueWriter := &style.IssueCollector{Issues: &fileIssues}
				command.Commands["style"].ExecuteWithRules(nodes, file, issueWriter, c.Rules)
				allIssues = append(allIssues, fileIssues...)
			}
			// Print grouped output and summary
			style.PrintPHPCSStyleOutputToWriter(outWriter, allIssues)
			totalParseErrors = 0 // Not tracked in streaming mode
			totalLines = 0      // Not tracked in streaming mode
		} else {
			totalParseErrors, totalLines = command.ProcessMultipleFiles(filesToScan, commandName, *debug, *parallelism, outWriter)
		}
	}

	trackMemoryUsage(&mem, false)
	elapsed := time.Since(start).Seconds()
	printSummary(outWriter, totalParseErrors, totalLines, elapsed, mem)
}
