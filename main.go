package main

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/config"
	"go-phpcs/style"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
	"go-phpcs/utils"
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
	profile := flag.Bool("profile", false, "Enable CPU and memory profiling (cpu.prof, mem.prof)")
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

	// Profiling support: only enabled if --profile is set
	if *profile {
		f, err := os.Create("cpu.prof")
		if err != nil {
			log.Fatalf("could not create CPU profile: %v", err)
		}
		pprof.StartCPUProfile(f)
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
			mf, err := os.Create("mem.prof")
			if err == nil {
				pprof.WriteHeapProfile(mf)
				mf.Close()
			}
		}()
	}

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
			totalParseErrors = 0
			totalLines = 0
			nFiles := len(filesToScan)
			progressBar := utils.NewProgressBar(nFiles, "Scanning")

			var processed int
			// Wrap the callback to update the progress bar for each file
			allIssues, totalParseErrors, totalLines = command.ProcessStyleFilesParallelWithCallback(filesToScan, c.Rules, *parallelism, func() {
				processed++
				progressBar.Print(processed)
			})
			style.PrintPHPCSStyleOutputToWriter(outWriter, allIssues)

		} else {
			totalParseErrors, totalLines = command.ProcessMultipleFiles(filesToScan, commandName, *debug, *parallelism, outWriter)
		}
	}

	trackMemoryUsage(&mem, false)
	elapsed := time.Since(start).Seconds()
	printSummary(outWriter, totalParseErrors, totalLines, elapsed, mem)
}

// repeat returns a string consisting of n copies of the rune r
func repeat(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}


// isatty returns true if the given file descriptor is a terminal
func isatty(fd uintptr) bool {
	// Only works on unix-like systems; for cross-platform use, consider a third-party lib
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

