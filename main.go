package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/config"
	"log"
	"os"
	"runtime"
	"time"
)

const errorLineFormat = "\t%s\n"

type memStats struct {
	start, end runtime.MemStats
}

func printSummary(totalParseErrors, totalLines int, elapsed float64, mem memStats) {
	fmt.Printf("\nScan completed in %.2f seconds\n", elapsed)
	if elapsed > 0 {
		fmt.Printf("Total lines scanned: %d\n", totalLines)
		fmt.Printf("Lines per second: %.2f\n", float64(totalLines)/elapsed)
	} else {
		fmt.Printf("Total lines scanned: %d\n", totalLines)
		fmt.Printf("Lines per second: N/A (too fast to measure)\n")
	}
	fmt.Printf("Total parsing errors: %d\n", totalParseErrors)
	fmt.Printf("HeapAlloc: %.2f MB\n", float64(mem.end.HeapAlloc)/(1024*1024))
	fmt.Printf("Sys: %.2f MB\n", float64(mem.end.Sys)/(1024*1024))
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
	var mem memStats
	trackMemoryUsage(&mem, true)

	totalParseErrors := 0
	totalLines := 0

	if len(flag.Args()) > 1 {
		filePath := flag.Args()[1]
		if filePath == "" {
			fmt.Println("No file specified for parsing.")
			command.PrintUsage()
			os.Exit(1)
		}
		totalParseErrors, totalLines = command.ProcessSingleFile(filePath, commandName, *debug)
	} else {
		if len(filesToScan) == 0 {
			fmt.Println("No files to scan.")
			os.Exit(1)
		}
		totalParseErrors, totalLines = command.ProcessMultipleFiles(filesToScan, commandName, *debug, *parallelism)
	}

	trackMemoryUsage(&mem, false)
	elapsed := time.Since(start).Seconds()
	printSummary(totalParseErrors, totalLines, elapsed, mem)
}
