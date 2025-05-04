package helper

import (
	"flag"
	"fmt"
	"go-phpcs/command"
	"go-phpcs/config"
	"go-phpcs/style"
	"go-phpcs/utils"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

type CliArgs struct {
	Profile         bool
	CommandName     string
	outputFile      string
	outputFileShort string
	debug           bool
	parallelism     int
	filePath        string
	Fix             bool
}

func ParseCLIArgs(filesToScan []string) CliArgs {
	profile := flag.Bool("profile", false, "Enable CPU and memory profiling (cpu.prof, mem.prof)")
	outputFile := flag.String("output", "", "Write all output (including summary) to this file")
	outputFileShort := flag.String("o", "", "Write all output (including summary) to this file (shorthand)")
	debug := flag.Bool("debug", false, "Enable debug mode to show parsing errors")
	parallelism := flag.Int("p", 2, "Number of files to process in parallel (default 2 for memory efficiency)")
	fix := flag.Bool("fix", false, "Automatically fix fixable style issues")
	flag.Parse()

	commandName := "style"
	if len(flag.Args()) > 0 {
		commandName = flag.Args()[0]
	}
	filePath := ""
	if len(flag.Args()) > 1 {
		filePath = flag.Args()[1]
	}
	return CliArgs{
		Profile:         *profile,
		CommandName:     commandName,
		outputFile:      *outputFile,
		outputFileShort: *outputFileShort,
		debug:           *debug,
		parallelism:     *parallelism,
		filePath:        filePath,
		Fix:             *fix,
	}
}

func SetupOutputFile(args CliArgs) io.Writer {
	if args.outputFile != "" {
		f, err := os.Create(args.outputFile)
		if err != nil {
			log.Fatalf("Could not create output file %s: %v", args.outputFile, err)
		}
		return f
	} else if args.outputFileShort != "" {
		f, err := os.Create(args.outputFileShort)
		if err != nil {
			log.Fatalf("Could not create output file %s: %v", args.outputFileShort, err)
		}
		return f
	}
	return os.Stdout
}

func SetupProfiling(enabled bool) func() {
	if !enabled {
		return func() {}
	}
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatalf("could not create CPU profile: %v", err)
	}
	pprof.StartCPUProfile(f)
	return func() {
		pprof.StopCPUProfile()
		f.Close()
		mf, err := os.Create("mem.prof")
		if err == nil {
			pprof.WriteHeapProfile(mf)
			mf.Close()
		}
	}
}

type MemStats struct {
	Start, End runtime.MemStats
}

func TrackMemoryUsage(mem *MemStats, atStart bool) {
	if atStart {
		runtime.ReadMemStats(&mem.Start)
	} else {
		runtime.GC()
		runtime.ReadMemStats(&mem.End)
	}
}

func RunScanOrCommand(args CliArgs, c *config.Config, filesToScan []string, outWriter io.Writer, mem *MemStats) (int, int) {
	totalParseErrors := 0
	totalLines := 0
	if args.filePath != "" {
		totalParseErrors, totalLines = command.ProcessFile(args.filePath, args.CommandName, args.debug, outWriter), 0
	} else {
		if len(filesToScan) == 0 {
			fmt.Fprintln(outWriter, "No files to scan.")
			os.Exit(1)
		}
		if args.CommandName == "style" {
			var allIssues []style.StyleIssue
			nFiles := len(filesToScan)
			progressBar := utils.NewProgressBar(nFiles, "Scanning")
			var processed int
			allIssues, totalParseErrors, totalLines = command.ProcessStyleFilesParallelWithCallback(filesToScan, c.Rules, args.parallelism, func() {
				processed++
				progressBar.Print(processed)
			})

			if args.Fix {
				// Group issues by file
				fileToIssues := map[string][]style.StyleIssue{}
				for _, iss := range allIssues {
					if iss.Fixable {
						fileToIssues[iss.Filename] = append(fileToIssues[iss.Filename], iss)
					}
				}
				for file, issues := range fileToIssues {
					input, err := os.ReadFile(file)
					if err != nil {
						fmt.Fprintf(outWriter, "[fix] Could not read file %s: %v\n", file, err)
						continue
					}
					content := string(input)
					applied := map[string]bool{}
					for _, iss := range issues {
						if applied[iss.Code] {
							continue
						} // Only apply each fix once per file
						fixer := style.GetFixer(iss.Code)
						if fixer != nil {
							content = fixer.Fix(content)
							applied[iss.Code] = true
						}
					}
					err = os.WriteFile(file, []byte(content), 0644)
					if err != nil {
						fmt.Fprintf(outWriter, "[fix] Could not write file %s: %v\n", file, err)
					} else {
						fmt.Fprintf(outWriter, "[fix] Applied fixes to %s\n", file)
					}
				}
			}

			fmt.Fprintln(outWriter, "\033[36;1m\n========== SCAN RESULTS =========="+"\033[0m")
			style.PrintPHPCSStyleOutputToWriter(outWriter, allIssues)
		} else {
			totalParseErrors, totalLines = command.ProcessMultipleFiles(filesToScan, args.CommandName, args.debug, args.parallelism, outWriter)
		}
	}
	return totalParseErrors, totalLines
}

func PrintSummary(w io.Writer, totalParseErrors, totalLines int, elapsed float64, mem MemStats) {
	fmt.Fprintln(w, "\033[36;1m\n========== PERFORMANCE METRICS =========="+"\033[0m")
	if elapsed > 0 {
		fmt.Fprintf(w, "Total lines scanned: \033[32;1m%d\033[0m\n", totalLines)
		fmt.Fprintf(w, "Lines per second: \033[32;1m%.2f\033[0m\n", float64(totalLines)/elapsed)
	} else {
		fmt.Fprintf(w, "Total lines scanned: \033[32;1m%d\033[0m\n", totalLines)
		fmt.Fprintf(w, "Lines per second: N/A (too fast to measure)\n")
	}
	fmt.Fprintf(w, "Total parsing errors: \033[31;1m%d\033[0m\n", totalParseErrors)
	fmt.Fprintf(w, "HeapAlloc: \033[35m%.2f MB\033[0m\n", float64(mem.End.HeapAlloc)/(1024*1024))
	fmt.Fprintf(w, "Sys: \033[35m%.2f MB\033[0m\n", float64(mem.End.Sys)/(1024*1024))
}
