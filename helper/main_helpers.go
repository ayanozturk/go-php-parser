package helper

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ayanozturk/go-php-parser/command"
	"github.com/ayanozturk/go-php-parser/config"
	"github.com/ayanozturk/go-php-parser/overrides"
	"github.com/ayanozturk/go-php-parser/style"
	"github.com/ayanozturk/go-php-parser/utils"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"slices"
	"sort"
)

type CliArgs struct {
	Profile         bool
	CommandName     string
	ConfigPath      string
	outputFile      string
	outputFileShort string
	debug           bool
	parallelism     int
	filePath        string
	Fix             bool
	PprofAddr       string
}

func (args CliArgs) HasExplicitFile() bool {
	return args.filePath != ""
}

func ParseCLIArgs(filesToScan []string) CliArgs {
	configPath := flag.String("config", "", "Path to config file (default: discover go-phpcs.yaml, go-phpcs.yml, config.yaml)")
	profile := flag.Bool("profile", false, "Enable CPU and memory profiling (cpu.prof, mem.prof)")
	outputFile := flag.String("output", "", "Write all output (including summary) to this file")
	outputFileShort := flag.String("o", "", "Write all output (including summary) to this file (shorthand)")
	debug := flag.Bool("debug", false, "Enable debug mode to show parsing errors")
	parallelism := flag.Int("p", 0, "Number of files to process in parallel (0=auto: NumCPU)")
	fix := flag.Bool("fix", false, "Automatically fix fixable style issues")
	pprofAddr := flag.String("pprof", "", "Start pprof HTTP server on addr (e.g. localhost:6060)")
	flag.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Printf("pprof listening on http://%s/debug/pprof/", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Printf("pprof: %v", err)
			}
		}()
	}

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
		ConfigPath:      *configPath,
		outputFile:      *outputFile,
		outputFileShort: *outputFileShort,
		debug:           *debug,
		parallelism: func() int {
			if *parallelism <= 0 {
				return runtime.NumCPU()
			}
			return *parallelism
		}(),
		filePath:  filePath,
		Fix:       *fix,
		PprofAddr: *pprofAddr,
	}
}

func SetupOutputFile(args CliArgs) io.Writer {
	if args.outputFile != "" {
		f, err := os.Create(args.outputFile)
		if err != nil {
			log.Fatalf("Could not create output file %s: %v", args.outputFile, err)
		}
		return bufio.NewWriterSize(f, 256*1024)
	} else if args.outputFileShort != "" {
		f, err := os.Create(args.outputFileShort)
		if err != nil {
			log.Fatalf("Could not create output file %s: %v", args.outputFileShort, err)
		}
		return bufio.NewWriterSize(f, 256*1024)
	}
	return os.Stdout
}

func PrintFileList(w io.Writer, files []string) {
	if len(files) == 0 {
		fmt.Fprintln(w, "No files selected by config.")
		return
	}

	sortedFiles := append([]string(nil), files...)
	sort.Strings(sortedFiles)
	for _, file := range sortedFiles {
		fmt.Fprintln(w, file)
	}
}

func SetupProfiling(enabled bool) func() {
	if !enabled {
		return func() {
			// Profiling is disabled, so return a no-op cleanup function.
		}
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

type RunOutcome struct {
	TotalParseErrors int
	TotalLines       int
	Diagnostics      int
	ExitCode         int
}

func TrackMemoryUsage(mem *MemStats, atStart bool) {
	if atStart {
		runtime.ReadMemStats(&mem.Start)
	} else {
		// Avoid runtime.GC() here — on large codebases the forced GC stalls for
		// minutes tracing pinned AST-substring data. Stats are slightly optimistic
		// (dead objects not yet collected) but the scan timing is correct.
		runtime.ReadMemStats(&mem.End)
	}
}

func RunScanOrCommand(args CliArgs, c *config.Config, filesToScan []string, outWriter io.Writer, mem *MemStats) RunOutcome {
	outcome := RunOutcome{}
	matcher, err := overrides.Compile(c.Overrides)
	if err != nil {
		fmt.Fprintf(outWriter, "Error compiling overrides: %v\n", err)
		outcome.ExitCode = 2
		return outcome
	}
	command.ConfigureAnalysis(c.AnalysisLevel)
	if args.CommandName == "analyze" {
		// Resolve the user's analyze target. A bare `analyze` reports on the
		// whole configured project; `analyze <file>` reports on a single
		// file; `analyze <folder>` reports on every PHP file under that
		// folder (respecting config.Extensions and config.Ignore). In all
		// three cases the whole project is still indexed for cross-file
		// symbol resolution.
		targets, err := resolveAnalyzeTargets(args.filePath, c, outWriter)
		if err != nil {
			fmt.Fprintf(outWriter, "Error resolving analyze target: %v\n", err)
			outcome.ExitCode = 2
			return outcome
		}
		if args.HasExplicitFile() && len(targets) == 0 {
			command.PrintAnalyzeResult(outWriter, command.AnalyzeResult{})
			return outcome
		}

		reportable := filesToScan
		for _, t := range targets {
			if !slices.Contains(reportable, t) {
				reportable = append(reportable, t)
			}
		}

		// config.Includes adds extra directories (e.g. vendor) that are
		// indexed for symbol resolution but never reported on.
		includeFiles, err := config.GetIncludeFiles(c)
		if err != nil {
			fmt.Fprintf(outWriter, "Error scanning includes: %v\n", err)
			outcome.ExitCode = 2
			return outcome
		}
		reportableSet := make(map[string]struct{}, len(reportable))
		for _, f := range reportable {
			reportableSet[f] = struct{}{}
		}
		files := reportable
		for _, f := range includeFiles {
			if _, ok := reportableSet[f]; !ok {
				files = append(files, f)
			}
		}

		// Only run the expensive per-file analysis (CFGs, type inference,
		// narrowing, rule execution) against what we'll actually report:
		// the requested file(s), or every path-scanned file when running
		// whole-project (config.Includes files stay index-only).
		analysisTargets := targets
		if len(analysisTargets) == 0 {
			analysisTargets = reportable
		}

		// Use incremental analysis with cache in user's home directory
		cacheDir := ""
		homeDir, err := os.UserHomeDir()
		if err == nil {
			cacheDir = filepath.Join(homeDir, ".cache", "go-phpcs")
		}
		result := command.AnalyzeFilesIncrementalScoped(files, analysisTargets, c.AnalysisLevel, matcher, args.parallelism, cacheDir)
		if len(targets) > 0 {
			targetSet := make(map[string]struct{}, len(targets))
			for _, t := range targets {
				targetSet[t] = struct{}{}
			}
			result = command.FilterAnalyzeResultToFiles(result, targetSet)
		} else if len(includeFiles) > 0 {
			result = command.FilterAnalyzeResultToFiles(result, reportableSet)
		}
		command.PrintAnalyzeResult(outWriter, result)
		outcome.TotalParseErrors = countCommandParseErrors(result.ParseErrors)
		outcome.TotalLines = result.TotalLines
		outcome.Diagnostics = len(result.Issues)
		switch {
		case len(result.ReadErrors) > 0:
			outcome.ExitCode = 2
		case outcome.TotalParseErrors > 0 || outcome.Diagnostics > 0:
			outcome.ExitCode = 1
		}
		return outcome
	}
	if args.filePath != "" {
		errList, lineCount := command.ProcessFileWithErrors(args.filePath, args.CommandName, args.debug, c.Rules, matcher, outWriter)
		outcome.TotalParseErrors = len(errList)
		outcome.TotalLines = lineCount
		if len(errList) > 0 {
			fmt.Fprintf(outWriter, "Parsing errors in %s (%d error(s)):\n", args.filePath, len(errList))
			for _, err := range errList {
				fmt.Fprintf(outWriter, command.ErrorLineFormat, err)
			}
		}
	} else {
		if len(filesToScan) == 0 {
			fmt.Fprintln(outWriter, "No files to scan.")
			outcome.ExitCode = 2
			return outcome
		}
		if args.CommandName == "style" {
			var allIssues []style.StyleIssue
			nFiles := len(filesToScan)
			progressBar := utils.NewProgressBar(nFiles, "Scanning")
			var processed int
			allIssues, outcome.TotalParseErrors, outcome.TotalLines = command.ProcessStyleFilesParallelWithCallback(filesToScan, c.Rules, matcher, args.parallelism, func() {
				processed++
				progressBar.Print(processed)
			})

			if args.Fix {
				// Group issues by file
				fileToIssues := map[string][]style.StyleIssue{}
				var appliedFixes []style.StyleIssue
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
							appliedFixes = append(appliedFixes, iss)
						}
					}
					err = os.WriteFile(file, []byte(content), 0644)
					if err != nil {
						fmt.Fprintf(outWriter, "[fix] Could not write file %s: %v\n", file, err)
					} else {
						fmt.Fprintf(outWriter, "[fix] Applied fixes to %s\n", file)
					}
				}
				fmt.Fprintf(outWriter, "\n\033[36;1mFixed %d issue(s).\033[0m\n", len(appliedFixes))
				return outcome
			}

			fmt.Fprintln(outWriter, "\033[36;1m\n========== SCAN RESULTS =========="+"\033[0m")
			style.PrintPHPCSStyleOutputToWriter(outWriter, allIssues)
		} else {
			outcome.TotalParseErrors, outcome.TotalLines = command.ProcessMultipleFiles(filesToScan, args.CommandName, args.debug, args.parallelism, c.Rules, matcher, outWriter)
		}
	}
	return outcome
}

// resolveAnalyzeTargets turns the positional argument passed to the
// `analyze` command into a concrete list of PHP files to report on.
//
//   - An empty arg means "analyze the whole configured project" and
//     returns an empty slice (caller treats that as "no override").
//   - A file path returns a one-element slice.
//   - A directory path is walked with the same extension and ignore
//     rules used by config.GetFilesToScan so behavior matches a
//     hand-written config pointing at the same folder.
//
// A non-existent path returns an error so the caller can surface a
// useful message instead of producing a silently empty result.
func resolveAnalyzeTargets(path string, c *config.Config, outWriter io.Writer) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("could not stat %q: %w", path, err)
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	ignoreDirs := make(map[string]struct{}, len(c.Ignore))
	for _, d := range c.Ignore {
		ignoreDirs[d] = struct{}{}
	}
	allowedExts := make(map[string]struct{}, len(c.Extensions))
	for _, ext := range c.Extensions {
		allowedExts["."+ext] = struct{}{}
	}
	var files []string
	walkErr := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, ignored := ignoreDirs[d.Name()]; ignored {
				return filepath.SkipDir
			}
			return nil
		}
		if _, allowed := allowedExts[filepath.Ext(p)]; allowed {
			files = append(files, p)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walking %q: %w", path, walkErr)
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(outWriter, "No analyzable files found under %q (extensions=%v, ignore=%v).\n", path, c.Extensions, c.Ignore)
	}
	return files, nil
}

func countCommandParseErrors(details []command.ParseErrorDetail) int {
	total := 0
	for _, detail := range details {
		total += len(detail.Errors)
	}
	return total
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
