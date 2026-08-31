// Command benchmark is the checked-in cold/full/incremental benchmark
// harness required by docs/full-static-analyser-target.md's M0 exit
// criteria ("Add the external three-project benchmark harness ... Measure
// index-only, cold full analysis, warm full analysis, and incremental edits
// separately ... Account for every discovered file and classify parser
// failures ... Record rule coverage and diagnostic counts beside timing and
// RSS.").
//
// It exercises the same engine as the production analyser
// (github.com/ayanozturk/go-php-parser/analyse), not the style checker, so
// results reflect the full-analyser target rather than PSR-12 style rules.
//
// Usage:
//
//	go run ./cmd/benchmark --root test_projects/symfony --json
//
// Index-only and cold-full-analysis measurements each re-exec this binary
// as a fresh subprocess per measured run, so every measured run starts with
// empty in-process caches (parsedTypeCache, project index, OS process
// state). Worker subprocesses pin GOMAXPROCS to --workers. After an
// unmeasured validation run the harness discards --cold-warmups process-cold
// subprocesses, pauses --settle-ms between measured runs, and may append
// --extra-cold-runs when the CV gate fails while still evaluating every
// measured sample. The parent measures the entire child lifetime, including startup,
// discovery, reads, parsing, indexing, analysis, reduction, and result
// serialization — a "process-cold" run in the terms of the comparable-
// performance contract. It does NOT drop the OS page/file cache; callers
// that need a filesystem-cold measurement too must do that themselves (e.g. `purge` on
// macOS, `echo 3 > /proc/sys/vm/drop_caches` on Linux) before invoking this
// harness, and should record which variant a given report reflects.
//
// Warm-full-analysis is measured by looping the full pipeline inside a
// single worker process: source files are read and parsed once, then
// project-index construction and rule analysis are repeated so the
// process's own caches and the OS file cache are warm for every measured
// iteration after the first (unmeasured) warmup iteration.
//
// Incremental-edit timing is not implemented yet: the analysis engine has
// no incremental invalidation API (immutable project graph / incremental
// analysis are M1/M2 roadmap items), so this harness reports
// "incremental": {"supported": false} rather than a fabricated number.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

// runMetrics captures everything the comparable-performance contract asks
// for a single measured run: timing, file accounting, and diagnostic
// counts. RSS is filled in by the harness (from OS rusage) for subprocess
// runs, and by the worker itself (from runtime.MemStats) as a same-process
// fallback for the warm-loop phase, where there is no child process to ask.
type runMetrics struct {
	DurationMs         int64 `json:"durationMs"`
	FilesDiscovered    int   `json:"filesDiscovered"`
	FilesParsed        int   `json:"filesParsed"`
	FilesFailed        int   `json:"filesFailed"`
	TotalLOC           int   `json:"totalLoc"`
	TotalBytes         int64 `json:"totalBytes"`
	DiagnosticsEmitted int   `json:"diagnosticsEmitted"`
	GoMemSysPeakBytes  int64 `json:"goMemSysPeakBytes"`
	PeakRSSBytes       int64 `json:"peakRssBytes,omitempty"`
}

// phaseReport aggregates N measured runs of the same phase per the
// contract: run count, mean, median, min, max, standard deviation, and
// coefficient of variation, plus the file/diagnostic accounting from the
// last run (all runs scan the same corpus, so counts should be stable; any
// divergence between runs is itself a signal worth surfacing, so raw runs
// are kept in full rather than only the aggregate).
type phaseReport struct {
	Runs                    []runMetrics `json:"runs"`
	MeanMs                  float64      `json:"meanMs"`
	MedianMs                float64      `json:"medianMs"`
	MinMs                   float64      `json:"minMs"`
	MaxMs                   float64      `json:"maxMs"`
	StdDevMs                float64      `json:"stdDevMs"`
	CoefficientOfVar        float64      `json:"coefficientOfVariation"`
	CoefficientOfVarDropMax float64      `json:"coefficientOfVariationDropMax,omitempty"`
	MaxPeakRSSBytes         int64        `json:"maxPeakRssBytes,omitempty"`
}

type incrementalReport struct {
	Supported bool   `json:"supported"`
	Reason    string `json:"reason,omitempty"`
}

type benchmarkRunTarget string

const (
	benchmarkCandidate benchmarkRunTarget = "candidate"
	benchmarkBaseline  benchmarkRunTarget = "baseline"
)

type baselineReport struct {
	Binary           string      `json:"binary"`
	ValidationRun    runMetrics  `json:"validationRun"`
	ColdFullAnalysis phaseReport `json:"coldFullAnalysis"`
}

type benchmarkValidation struct {
	Accepted          bool     `json:"accepted"`
	MaxCV             float64  `json:"maxCoefficientOfVariation"`
	ExtraColdRunsUsed int      `json:"extraColdRunsUsed,omitempty"`
	Reasons           []string `json:"reasons,omitempty"`
}

type benchmarkReport struct {
	GeneratedAt   string              `json:"generatedAt"`
	Root          string              `json:"root"`
	GoVersion     string              `json:"goVersion"`
	OS            string              `json:"os"`
	Arch          string              `json:"arch"`
	NumCPU        int                 `json:"numCpu"`
	Workers       int                 `json:"workers"`
	Host          hostEnvironment     `json:"host"`
	Level         *int                `json:"analysisLevel"`
	Paths         []string            `json:"paths"`
	Excludes      []string            `json:"excludes,omitempty"`
	ValidationRun runMetrics          `json:"validationRun"`
	Baseline      *baselineReport     `json:"baseline,omitempty"`
	Validation    benchmarkValidation `json:"validation"`

	IndexOnly        phaseReport       `json:"indexOnly"`
	ColdFullAnalysis phaseReport       `json:"coldFullAnalysis"`
	WarmFullAnalysis phaseReport       `json:"warmFullAnalysis"`
	Incremental      incrementalReport `json:"incremental"`
}

func main() {
	root := flag.String("root", "test_projects", "root directory of the PHP corpus to analyse")
	pathsFlag := flag.String("paths", ".", "comma-separated paths within root to scan (for example src,tests,vendor)")
	excludesFlag := flag.String("excludes", "", "comma-separated paths within root to exclude")
	level := flag.Int("level", -1, "analysis level filter to apply (-1 = run every registered rule, matching the default config)")
	workers := flag.Int("workers", runtime.NumCPU(), "worker goroutines for parsing/analysis within a single run")
	coldRuns := flag.Int("cold-runs", 10, "number of measured process-cold full-analysis runs (contract minimum is 10)")
	warmIterations := flag.Int("warm-iterations", 11, "in-process warm-loop iterations; the first is an unmeasured warmup")
	jsonOutput := flag.Bool("json", false, "emit JSON instead of a text summary")
	outputPath := flag.String("output", "", "optional file to write the report to")
	skipCold := flag.Bool("skip-cold", false, "skip the process-cold subprocess runs (index-only and warm-loop still run)")
	baselineBinary := flag.String("baseline-binary", "", "optional previous-engine benchmark binary to interleave with candidate cold runs")
	maxCV := flag.Float64("max-cv", 0.05, "maximum accepted cold-run coefficient of variation (0 disables the gate)")
	settleMs := flag.Int("settle-ms", 250, "pause between process-cold subprocesses so frequency scaling and background load can settle")
	coldWarmups := flag.Int("cold-warmups", 1, "unmeasured process-cold full-analysis subprocesses per engine after validation and before measured runs")
	extraColdRuns := flag.Int("extra-cold-runs", 10, "additional measured runs per engine when the CV gate fails; 0 disables extension. The gate still uses every measured sample.")

	// Internal re-exec entrypoint: when set, this process performs exactly
	// one measured phase and prints its runMetrics as a single JSON line to
	// stdout, instead of acting as the harness. Not intended for direct use.
	workerPhase := flag.String("harness-worker-phase", "", "internal: run a single worker phase (index|full) and exit")
	workerRepeat := flag.Int("harness-worker-repeat", 1, "internal: warm-loop iteration count for the full phase")

	cpuProfile := flag.String("cpuprofile", "", "write a CPU profile (pprof format) of a single in-process full-analysis run to this path, then exit without running the benchmark suite")
	memProfile := flag.String("memprofile", "", "write a heap profile (pprof format) after a single in-process full-analysis run to this path, then exit without running the benchmark suite")
	profileIterations := flag.Int("profile-iterations", 1, "number of in-process full-analysis iterations to run while profiling (first iteration includes one-time setup costs like parsing)")

	flag.Parse()
	paths, err := parseBenchmarkPaths(*pathsFlag, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark: invalid --paths: %v\n", err)
		os.Exit(1)
	}
	excludes, err := parseBenchmarkPaths(*excludesFlag, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark: invalid --excludes: %v\n", err)
		os.Exit(1)
	}

	if *workerPhase != "" {
		runWorker(*workerPhase, *root, paths, excludes, *level, *workers, *workerRepeat)
		return
	}

	if *cpuProfile != "" || *memProfile != "" {
		if err := runProfile(*root, paths, excludes, *level, *workers, *profileIterations, *cpuProfile, *memProfile); err != nil {
			fmt.Fprintf(os.Stderr, "benchmark: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *workers < 1 {
		*workers = 1
	}
	if *coldRuns < 1 {
		*coldRuns = 1
	}
	if *warmIterations < 2 {
		*warmIterations = 2
	}

	var levelPtr *int
	if *level >= 0 {
		levelPtr = level
	}

	report := benchmarkReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Root:        *root,
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		NumCPU:      runtime.NumCPU(),
		Workers:     *workers,
		Level:       levelPtr,
		Paths:       paths,
		Excludes:    excludes,
		Validation:  benchmarkValidation{Accepted: true, MaxCV: *maxCV},
		Incremental: incrementalReport{
			Supported: false,
			Reason:    "this CLI harness measures process-cold and warm full analysis only; editor-path incremental timing is gated by vscode-php-strom/cmd/benchmark-editor",
		},
	}
	if *coldWarmups < 0 {
		*coldWarmups = 0
	}
	if *extraColdRuns < 0 {
		*extraColdRuns = 0
	}
	if *settleMs < 0 {
		*settleMs = 0
	}
	report.Host = hostEnvironment{
		GoMaxProcs:         *workers,
		ProcessColdWarmups: *coldWarmups,
		SettleMs:           *settleMs,
		ExtraColdBudget:    *extraColdRuns,
	}
	if one, five, fifteen, ok := hostLoadAverages(); ok {
		report.Host.LoadAverage1 = one
		report.Host.LoadAverage5 = five
		report.Host.LoadAverage15 = fifteen
	}

	fmt.Fprintf(os.Stderr, "benchmark: discovering PHP files under %s (paths: %s)\n", *root, strings.Join(paths, ", "))
	indexRun, err := execWorker(*root, paths, excludes, *level, *workers, "index", 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark: index-only run failed: %v\n", err)
		os.Exit(1)
	}
	report.IndexOnly = summarize([]runMetrics{indexRun})

	fmt.Fprintln(os.Stderr, "benchmark: running unmeasured full-pipeline validation")
	validationRun, err := execWorker(*root, paths, excludes, *level, *workers, "full", 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark: validation run failed: %v\n", err)
		os.Exit(1)
	}
	report.ValidationRun = validationRun
	if err := validatePhaseAccounting(indexRun, []runMetrics{validationRun}, false); err != nil {
		report.Validation.Accepted = false
		report.Validation.Reasons = append(report.Validation.Reasons, "candidate validation: "+err.Error())
	}

	var baselineValidation runMetrics
	if *baselineBinary != "" {
		resolvedBaseline, resolveErr := filepath.Abs(*baselineBinary)
		if resolveErr != nil {
			fmt.Fprintf(os.Stderr, "benchmark: resolve baseline binary: %v\n", resolveErr)
			os.Exit(1)
		}
		if info, statErr := os.Stat(resolvedBaseline); statErr != nil || info.IsDir() {
			fmt.Fprintf(os.Stderr, "benchmark: invalid baseline binary %q\n", resolvedBaseline)
			os.Exit(1)
		}
		*baselineBinary = resolvedBaseline
		fmt.Fprintln(os.Stderr, "benchmark: running unmeasured baseline full-pipeline validation")
		baselineValidation, err = execWorkerBinary(*baselineBinary, *root, paths, excludes, *level, *workers, "full", 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchmark: baseline validation run failed: %v\n", err)
			os.Exit(1)
		}
		report.Baseline = &baselineReport{Binary: *baselineBinary, ValidationRun: baselineValidation}
		if err := validatePhaseAccounting(validationRun, []runMetrics{baselineValidation}, false); err != nil {
			report.Validation.Accepted = false
			report.Validation.Reasons = append(report.Validation.Reasons, "baseline validation: "+err.Error())
		}
	}

	if !*skipCold {
		fmt.Fprintf(os.Stderr, "benchmark: running %d unmeasured process-cold warmup(s) per engine\n", *coldWarmups)
		for i := 0; i < *coldWarmups; i++ {
			if _, err := execWorker(*root, paths, excludes, *level, *workers, "full", 1); err != nil {
				fmt.Fprintf(os.Stderr, "benchmark: candidate warmup failed: %v\n", err)
				os.Exit(1)
			}
			settle(*settleMs)
			if report.Baseline != nil {
				if _, err := execWorkerBinary(report.Baseline.Binary, *root, paths, excludes, *level, *workers, "full", 1); err != nil {
					fmt.Fprintf(os.Stderr, "benchmark: baseline warmup failed: %v\n", err)
					os.Exit(1)
				}
				settle(*settleMs)
			}
		}

		fmt.Fprintf(os.Stderr, "benchmark: running %d process-cold full-analysis runs\n", *coldRuns)
		coldRunsResults := make([]runMetrics, 0, *coldRuns+*extraColdRuns)
		baselineRunsResults := make([]runMetrics, 0, *coldRuns+*extraColdRuns)
		runCold := func(target benchmarkRunTarget) error {
			binary := ""
			if target == benchmarkBaseline {
				binary = report.Baseline.Binary
			}
			run, err := execWorkerBinary(binary, *root, paths, excludes, *level, *workers, "full", 1)
			if err != nil {
				return fmt.Errorf("%s cold run: %w", target, err)
			}
			if target == benchmarkBaseline {
				baselineRunsResults = append(baselineRunsResults, run)
				fmt.Fprintf(os.Stderr, "  baseline cold run %d: %dms, %d diagnostics\n", len(baselineRunsResults), run.DurationMs, run.DiagnosticsEmitted)
			} else {
				coldRunsResults = append(coldRunsResults, run)
				fmt.Fprintf(os.Stderr, "  candidate cold run %d: %dms, %d diagnostics\n", len(coldRunsResults), run.DurationMs, run.DiagnosticsEmitted)
			}
			settle(*settleMs)
			return nil
		}

		orders := make([]benchmarkRunTarget, *coldRuns)
		for i := range orders {
			orders[i] = benchmarkCandidate
		}
		if report.Baseline != nil {
			orders = interleavedRunOrder(*coldRuns)
		}
		for _, target := range orders {
			if err := runCold(target); err != nil {
				fmt.Fprintf(os.Stderr, "benchmark: %v\n", err)
				os.Exit(1)
			}
		}

		for extraUsed := 0; extraUsed < *extraColdRuns; extraUsed++ {
			report.ColdFullAnalysis = summarize(coldRunsResults)
			var baselinePhase *phaseReport
			if report.Baseline != nil {
				summarized := summarize(baselineRunsResults)
				report.Baseline.ColdFullAnalysis = summarized
				baselinePhase = &summarized
			}
			if !shouldExtendColdRuns(report.ColdFullAnalysis, baselinePhase, *maxCV, *extraColdRuns-extraUsed) {
				break
			}
			fmt.Fprintf(os.Stderr, "benchmark: extending measured cold runs (%d/%d extra)\n", extraUsed+1, *extraColdRuns)
			if report.Baseline != nil {
				for _, target := range extraInterleavedPair(len(coldRunsResults)) {
					if err := runCold(target); err != nil {
						fmt.Fprintf(os.Stderr, "benchmark: %v\n", err)
						os.Exit(1)
					}
				}
			} else if err := runCold(benchmarkCandidate); err != nil {
				fmt.Fprintf(os.Stderr, "benchmark: %v\n", err)
				os.Exit(1)
			}
			report.Validation.ExtraColdRunsUsed++
		}

		report.ColdFullAnalysis = summarize(coldRunsResults)
		if err := validatePhaseAccounting(validationRun, coldRunsResults, true); err != nil {
			report.Validation.Accepted = false
			report.Validation.Reasons = append(report.Validation.Reasons, "candidate cold runs: "+err.Error())
		}
		if reason := validatePhaseCV("candidate", report.ColdFullAnalysis, *maxCV); reason != "" {
			report.Validation.Accepted = false
			report.Validation.Reasons = append(report.Validation.Reasons, reason)
		}
		if report.Baseline != nil {
			report.Baseline.ColdFullAnalysis = summarize(baselineRunsResults)
			if err := validatePhaseAccounting(baselineValidation, baselineRunsResults, true); err != nil {
				report.Validation.Accepted = false
				report.Validation.Reasons = append(report.Validation.Reasons, "baseline cold runs: "+err.Error())
			}
			if reason := validatePhaseCV("baseline", report.Baseline.ColdFullAnalysis, *maxCV); reason != "" {
				report.Validation.Accepted = false
				report.Validation.Reasons = append(report.Validation.Reasons, reason)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "benchmark: running warm-loop full-analysis (%d iterations, 1 unmeasured warmup)\n", *warmIterations)
	warmRuns, err := execWorkerRepeat(*root, paths, excludes, *level, *workers, *warmIterations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark: warm-loop run failed: %v\n", err)
		os.Exit(1)
	}
	// Drop the first (warmup) iteration from the measured set.
	report.WarmFullAnalysis = summarize(warmRuns[1:])

	out := io.Writer(os.Stdout)
	if *outputPath != "" {
		f, createErr := os.Create(*outputPath)
		if createErr != nil {
			fmt.Fprintf(os.Stderr, "benchmark: %v\n", createErr)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "benchmark: %v\n", err)
			os.Exit(1)
		}
		if !report.Validation.Accepted {
			os.Exit(2)
		}
		return
	}
	printTextReport(out, report)
	if !report.Validation.Accepted {
		os.Exit(2)
	}
}

// execWorker re-execs this binary as a fresh subprocess for exactly one
// measured phase, so package-level in-process caches never leak state
// between measured runs. It returns the worker's reported runMetrics with
// PeakRSSBytes overwritten by the OS-reported rusage of the child process,
// which is a more trustworthy peak-RSS source than in-process sampling.
func execWorker(root string, paths, excludes []string, level, workers int, phase string, repeat int) (runMetrics, error) {
	return execWorkerBinary("", root, paths, excludes, level, workers, phase, repeat)
}

func execWorkerBinary(binary, root string, paths, excludes []string, level, workers int, phase string, repeat int) (runMetrics, error) {
	self, err := os.Executable()
	if err != nil {
		return runMetrics{}, fmt.Errorf("resolving self path: %w", err)
	}
	if binary != "" {
		self = binary
	}
	args := []string{
		"--harness-worker-phase=" + phase,
		"--root=" + root,
		"--paths=" + strings.Join(paths, ","),
		"--excludes=" + strings.Join(excludes, ","),
		"--level=" + strconv.Itoa(level),
		"--workers=" + strconv.Itoa(workers),
		"--harness-worker-repeat=" + strconv.Itoa(repeat),
	}
	cmd := exec.Command(self, args...)
	cmd.Env = workerEnv(workers)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)
	if runErr != nil {
		return runMetrics{}, fmt.Errorf("worker exited with error: %w\nstderr: %s", runErr, stderr.String())
	}

	line := strings.TrimSpace(stdout.String())
	var result runMetrics
	if err := json.Unmarshal([]byte(line), &result); err != nil {
		return runMetrics{}, fmt.Errorf("parsing worker output %q: %w", line, err)
	}
	if rss := peakRSSBytes(cmd.ProcessState); rss > 0 {
		result.PeakRSSBytes = rss
	}
	// Measure process-cold phases at the parent boundary so startup, file
	// discovery, reads, parsing, indexing, analysis, reduction, and worker
	// result serialization are all inside the timed region. The worker's
	// internal duration is useful only for warm-loop iterations.
	result.DurationMs = duration.Milliseconds()
	return result, nil
}

func interleavedRunOrder(count int) []benchmarkRunTarget {
	if count <= 0 {
		return nil
	}
	order := make([]benchmarkRunTarget, 0, count*2)
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			order = append(order, benchmarkCandidate, benchmarkBaseline)
		} else {
			order = append(order, benchmarkBaseline, benchmarkCandidate)
		}
	}
	return order
}

func validatePhaseAccounting(reference runMetrics, runs []runMetrics, includeDiagnostics bool) error {
	for i, run := range runs {
		if run.FilesDiscovered != reference.FilesDiscovered || run.FilesParsed != reference.FilesParsed || run.FilesFailed != reference.FilesFailed || run.TotalLOC != reference.TotalLOC || run.TotalBytes != reference.TotalBytes {
			return fmt.Errorf("run %d file accounting differs from validation", i+1)
		}
		if includeDiagnostics && run.DiagnosticsEmitted != reference.DiagnosticsEmitted {
			return fmt.Errorf("run %d diagnostics = %d, validation = %d", i+1, run.DiagnosticsEmitted, reference.DiagnosticsEmitted)
		}
	}
	return nil
}

func validatePhaseCV(label string, phase phaseReport, maxCV float64) string {
	if maxCV <= 0 || len(phase.Runs) < 2 || phase.CoefficientOfVar <= maxCV {
		return ""
	}
	return fmt.Sprintf("%s cold CV %.4f exceeds %.4f", label, phase.CoefficientOfVar, maxCV)
}

// execWorkerRepeat re-execs a single subprocess that loops the full
// pipeline `repeat` times in-process, returning one runMetrics per
// iteration plus the process's OS-reported peak RSS attached to every
// iteration (a single process can only report one peak covering its whole
// lifetime, not a per-iteration breakdown).
func execWorkerRepeat(root string, paths, excludes []string, level, workers, repeat int) ([]runMetrics, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolving self path: %w", err)
	}
	args := []string{
		"--harness-worker-phase=full",
		"--root=" + root,
		"--paths=" + strings.Join(paths, ","),
		"--excludes=" + strings.Join(excludes, ","),
		"--level=" + strconv.Itoa(level),
		"--workers=" + strconv.Itoa(workers),
		"--harness-worker-repeat=" + strconv.Itoa(repeat),
	}
	cmd := exec.Command(self, args...)
	cmd.Env = workerEnv(workers)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr != nil {
		return nil, fmt.Errorf("worker exited with error: %w\nstderr: %s", runErr, stderr.String())
	}

	var results []runMetrics
	rss := peakRSSBytes(cmd.ProcessState)
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var result runMetrics
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			return nil, fmt.Errorf("parsing worker output %q: %w", line, err)
		}
		if rss > 0 {
			result.PeakRSSBytes = rss
		}
		results = append(results, result)
	}
	if len(results) != repeat {
		return nil, fmt.Errorf("expected %d warm-loop iterations, got %d", repeat, len(results))
	}
	return results, nil
}

// runWorker is the subprocess entrypoint invoked via --harness-worker-phase.
// It performs the requested phase against a freshly discovered file list and
// prints one JSON runMetrics line per iteration to stdout, then exits. All
// diagnostic/progress output must go to stderr so stdout stays a clean
// machine-readable stream for the harness to parse.
func runWorker(phase, root string, paths, excludes []string, level, workers, repeat int) {
	files, err := discoverPHPFiles(root, paths, excludes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark worker: %v\n", err)
		os.Exit(1)
	}

	var levelPtr *int
	if level >= 0 {
		levelPtr = &level
	}

	parsed, parseMetrics := parseFiles(files, workers)

	switch phase {
	case "index":
		peakSys := startMemSampler()
		start := time.Now()
		analyse.BuildProjectIndex(parsed)
		parseMetrics.DurationMs = time.Since(start).Milliseconds()
		parseMetrics.GoMemSysPeakBytes = peakSys()
		printResultLine(parseMetrics)
	case "full":
		if repeat < 1 {
			repeat = 1
		}
		for i := 0; i < repeat; i++ {
			// A fresh sampler per iteration: startMemSampler's stop channel
			// is one-shot, and each warm-loop iteration is its own
			// measured "run" that should report its own memory peak.
			peakSys := startMemSampler()
			start := time.Now()
			project := analyse.BuildProjectIndex(parsed)
			diagnostics := runAnalysis(parsed, project, levelPtr, workers)
			iter := parseMetrics
			iter.DurationMs = time.Since(start).Milliseconds()
			iter.DiagnosticsEmitted = diagnostics
			iter.GoMemSysPeakBytes = peakSys()
			printResultLine(iter)
		}
	default:
		fmt.Fprintf(os.Stderr, "benchmark worker: unknown phase %q\n", phase)
		os.Exit(1)
	}
}

// runProfile runs the full parse+index+analyse pipeline in-process (no
// subprocess re-exec, so pprof captures every allocation and CPU sample
// from the actual process being profiled) and writes CPU and/or heap
// profiles for offline analysis with `go tool pprof`. It intentionally
// bypasses the cold/warm harness measurement machinery: profiling wants
// the real work under a profiler attached to this process, not isolated
// subprocess timing.
func runProfile(root string, paths, excludes []string, level, workers, iterations int, cpuProfilePath, memProfilePath string) error {
	if iterations < 1 {
		iterations = 1
	}

	files, err := discoverPHPFiles(root, paths, excludes)
	if err != nil {
		return fmt.Errorf("discovering files: %w", err)
	}
	fmt.Fprintf(os.Stderr, "benchmark: profiling %d files under %s (%d iteration(s))\n", len(files), root, iterations)

	var levelPtr *int
	if level >= 0 {
		levelPtr = &level
	}

	if cpuProfilePath != "" {
		f, err := os.Create(cpuProfilePath)
		if err != nil {
			return fmt.Errorf("creating CPU profile %s: %w", cpuProfilePath, err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("starting CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	parsed, parseMetrics := parseFiles(files, workers)
	var diagnostics int
	for i := 0; i < iterations; i++ {
		start := time.Now()
		project := analyse.BuildProjectIndex(parsed)
		diagnostics = runAnalysis(parsed, project, levelPtr, workers)
		fmt.Fprintf(os.Stderr, "  iteration %d/%d: %s, %d diagnostics\n", i+1, iterations, time.Since(start), diagnostics)
	}
	fmt.Fprintf(os.Stderr, "benchmark: parsed %d/%d files, %d diagnostics on the final iteration\n", parseMetrics.FilesParsed, parseMetrics.FilesDiscovered, diagnostics)

	if memProfilePath != "" {
		f, err := os.Create(memProfilePath)
		if err != nil {
			return fmt.Errorf("creating heap profile %s: %w", memProfilePath, err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics, matching the standard pprof heap-profile recipe.
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("writing heap profile: %w", err)
		}
	}

	if cpuProfilePath != "" {
		fmt.Fprintf(os.Stderr, "benchmark: wrote CPU profile to %s (inspect with `go tool pprof %s %s`)\n", cpuProfilePath, os.Args[0], cpuProfilePath)
	}
	if memProfilePath != "" {
		fmt.Fprintf(os.Stderr, "benchmark: wrote heap profile to %s (inspect with `go tool pprof %s %s`)\n", memProfilePath, os.Args[0], memProfilePath)
	}
	return nil
}

func printResultLine(m runMetrics) {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(m); err != nil {
		fmt.Fprintf(os.Stderr, "benchmark worker: encoding result: %v\n", err)
		os.Exit(1)
	}
}

// startMemSampler launches a background goroutine that samples
// runtime.MemStats.Sys (Go's total memory obtained from the OS) at a short
// interval and tracks its peak. The returned function stops the sampler and
// returns the observed peak. This is an in-process fallback/supplement to
// OS rusage, useful on the warm-loop path where there is no child process
// boundary to read rusage from, and as a cross-check on subprocess runs.
func startMemSampler() func() int64 {
	stop := make(chan struct{})
	var peak int64
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		sample := func() {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			mu.Lock()
			if int64(ms.Sys) > peak {
				peak = int64(ms.Sys)
			}
			mu.Unlock()
		}
		sample()
		for {
			select {
			case <-stop:
				sample()
				return
			case <-ticker.C:
				sample()
			}
		}
	}()
	return func() int64 {
		close(stop)
		<-done
		mu.Lock()
		defer mu.Unlock()
		return peak
	}
}

func discoverPHPFiles(root string, paths, excludes []string) ([]string, error) {
	filesByPath := make(map[string]struct{})
	for _, relativeRoot := range paths {
		scanRoot := filepath.Join(root, filepath.FromSlash(relativeRoot))
		if _, err := os.Stat(scanRoot); err != nil {
			return nil, fmt.Errorf("scan path %q: %w", relativeRoot, err)
		}
		err := filepath.WalkDir(scanRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if benchmarkPathExcluded(relative, excludes) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".php") {
				filesByPath[path] = struct{}{}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	files := make([]string, 0, len(filesByPath))
	for path := range filesByPath {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func parseBenchmarkPaths(value string, allowEmpty bool) ([]string, error) {
	var paths []string
	seen := make(map[string]struct{})
	for _, raw := range strings.Split(value, ",") {
		path := filepath.ToSlash(filepath.Clean(strings.TrimSpace(raw)))
		if path == "." && strings.TrimSpace(raw) == "" {
			continue
		}
		if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
			return nil, fmt.Errorf("path %q must stay within the benchmark root", raw)
		}
		if _, duplicate := seen[path]; duplicate {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	if len(paths) == 0 && !allowEmpty {
		return nil, errors.New("at least one path is required")
	}
	sort.Strings(paths)
	return paths, nil
}

func benchmarkPathExcluded(path string, excludes []string) bool {
	for _, excluded := range excludes {
		if path == excluded || strings.HasPrefix(path, excluded+"/") {
			return true
		}
	}
	return false
}

// parseFiles reads and parses every discovered file concurrently, returning
// the successfully parsed ASTs keyed by path plus file-accounting metrics.
// Files that fail to read or that the parser reports errors for are counted
// as failed and excluded from project-index construction and analysis, but
// every discovered file is accounted for in FilesDiscovered per the
// contract's "account for every discovered file" requirement.
func parseFiles(files []string, workers int) (map[string][]ast.Node, runMetrics) {
	type parseOutcome struct {
		path    string
		nodes   []ast.Node
		loc     int
		bytes   int64
		failed  bool
		nodeErr error
	}

	pathCh := make(chan string, workers*2)
	resultCh := make(chan parseOutcome, workers*2)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathCh {
				content, err := os.ReadFile(path)
				if err != nil {
					resultCh <- parseOutcome{path: path, failed: true, nodeErr: err}
					continue
				}
				l := lexer.NewFile(string(content))
				p := parser.New(l, false)
				nodes := p.Parse()
				if len(p.Errors()) > 0 {
					resultCh <- parseOutcome{path: path, failed: true, bytes: int64(len(content))}
					continue
				}
				resultCh <- parseOutcome{
					path:  path,
					nodes: nodes,
					loc:   countLines(content),
					bytes: int64(len(content)),
				}
			}
		}()
	}
	go func() {
		for _, f := range files {
			pathCh <- f
		}
		close(pathCh)
	}()
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	parsed := make(map[string][]ast.Node, len(files))
	metrics := runMetrics{FilesDiscovered: len(files)}
	for outcome := range resultCh {
		if outcome.failed {
			metrics.FilesFailed++
			metrics.TotalBytes += outcome.bytes
			continue
		}
		parsed[outcome.path] = outcome.nodes
		metrics.FilesParsed++
		metrics.TotalLOC += outcome.loc
		metrics.TotalBytes += outcome.bytes
	}
	return parsed, metrics
}

func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	n := 1
	for _, b := range content {
		if b == '\n' {
			n++
		}
	}
	return n
}

// runAnalysis runs the registered analysis rules over every successfully
// parsed file concurrently against a shared project index — the same
// concurrency shape as command.ProcessStyleFilesParallelWithCallback, which
// is deliberate: this is the exact code path that a cyclic class hierarchy
// previously crashed via unbounded recursion in ancestor-walking helpers
// (see analyse/phpstan_level0_class_model.go), so this benchmark doubles as
// a regression exerciser for that fix at realistic corpus scale.
func runAnalysis(parsed map[string][]ast.Node, project *analyse.ProjectIndex, level *int, workers int) int {
	type job struct {
		path  string
		nodes []ast.Node
	}
	jobCh := make(chan job, workers*2)
	var total int64Counter
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				ctx := &analyse.AnalysisContext{Resolver: project, AnalysisLevel: level}
				issues := analyse.RunAnalysisRulesWithContext(j.path, j.nodes, ctx)
				total.add(int64(len(issues)))
			}
		}()
	}
	for path, nodes := range parsed {
		jobCh <- job{path: path, nodes: nodes}
	}
	close(jobCh)
	wg.Wait()
	return int(total.get())
}

// int64Counter is a tiny mutex-guarded counter; avoids pulling in
// sync/atomic call-site noise for a single accumulated total.
type int64Counter struct {
	mu    sync.Mutex
	value int64
}

func (c *int64Counter) add(n int64) {
	c.mu.Lock()
	c.value += n
	c.mu.Unlock()
}

func (c *int64Counter) get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func summarize(runs []runMetrics) phaseReport {
	if len(runs) == 0 {
		return phaseReport{}
	}
	durations := make([]float64, len(runs))
	var maxRSS int64
	for i, r := range runs {
		durations[i] = float64(r.DurationMs)
		if r.PeakRSSBytes > maxRSS {
			maxRSS = r.PeakRSSBytes
		}
	}
	sorted := append([]float64(nil), durations...)
	sort.Float64s(sorted)

	sum := 0.0
	for _, d := range durations {
		sum += d
	}
	mean := sum / float64(len(durations))

	var median float64
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		median = (sorted[mid-1] + sorted[mid]) / 2
	} else {
		median = sorted[mid]
	}

	variance := 0.0
	for _, d := range durations {
		diff := d - mean
		variance += diff * diff
	}
	variance /= float64(len(durations))
	stdDev := math.Sqrt(variance)

	cv := 0.0
	if mean != 0 {
		cv = stdDev / mean
	}

	dropMaxCV := 0.0
	if len(durations) >= 3 {
		dropMaxCV = coefficientOfVariation(dropOneMax(durations))
	}

	return phaseReport{
		Runs:                    runs,
		MeanMs:                  mean,
		MedianMs:                median,
		MinMs:                   sorted[0],
		MaxMs:                   sorted[len(sorted)-1],
		StdDevMs:                stdDev,
		CoefficientOfVar:        cv,
		CoefficientOfVarDropMax: dropMaxCV,
		MaxPeakRSSBytes:         maxRSS,
	}
}

func printTextReport(w io.Writer, r benchmarkReport) {
	fmt.Fprintf(w, "Full-Analyser Benchmark\n")
	fmt.Fprintf(w, "Root: %s\n", r.Root)
	fmt.Fprintf(w, "Generated: %s\n", r.GeneratedAt)
	fmt.Fprintf(w, "Go: %s  OS/Arch: %s/%s  CPUs: %d  Workers/GOMAXPROCS: %d\n", r.GoVersion, r.OS, r.Arch, r.NumCPU, r.Workers)
	fmt.Fprintf(w, "Host: coldWarmups=%d settleMs=%d extraColdBudget=%d extraColdUsed=%d", r.Host.ProcessColdWarmups, r.Host.SettleMs, r.Host.ExtraColdBudget, r.Validation.ExtraColdRunsUsed)
	if r.Host.LoadAverage1 > 0 || r.Host.LoadAverage5 > 0 || r.Host.LoadAverage15 > 0 {
		fmt.Fprintf(w, " loadavg=%.2f/%.2f/%.2f", r.Host.LoadAverage1, r.Host.LoadAverage5, r.Host.LoadAverage15)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Paths: %s", strings.Join(r.Paths, ", "))
	if len(r.Excludes) > 0 {
		fmt.Fprintf(w, "  Excludes: %s", strings.Join(r.Excludes, ", "))
	}
	fmt.Fprintln(w)
	if r.Level != nil {
		fmt.Fprintf(w, "Analysis level: %d\n\n", *r.Level)
	} else {
		fmt.Fprintf(w, "Analysis level: all registered rules\n\n")
	}

	printPhase := func(name string, p phaseReport) {
		if len(p.Runs) == 0 {
			fmt.Fprintf(w, "%s: (skipped)\n\n", name)
			return
		}
		last := p.Runs[len(p.Runs)-1]
		fmt.Fprintf(w, "%s (%d run(s)):\n", name, len(p.Runs))
		fmt.Fprintf(w, "  mean=%.1fms median=%.1fms min=%.1fms max=%.1fms stddev=%.1fms cv=%.3f",
			p.MeanMs, p.MedianMs, p.MinMs, p.MaxMs, p.StdDevMs, p.CoefficientOfVar)
		if p.CoefficientOfVarDropMax > 0 {
			fmt.Fprintf(w, " cvDropMax=%.3f", p.CoefficientOfVarDropMax)
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  filesDiscovered=%d filesParsed=%d filesFailed=%d totalLOC=%d totalBytes=%d diagnostics=%d\n",
			last.FilesDiscovered, last.FilesParsed, last.FilesFailed, last.TotalLOC, last.TotalBytes, last.DiagnosticsEmitted)
		if p.MaxPeakRSSBytes > 0 {
			fmt.Fprintf(w, "  peakRSS=%.1fMB (OS rusage)\n", float64(p.MaxPeakRSSBytes)/(1024*1024))
		}
		fmt.Fprintf(w, "  goMemSysPeak=%.1fMB (last run, in-process)\n\n", float64(last.GoMemSysPeakBytes)/(1024*1024))
	}

	printPhase("Index-only", r.IndexOnly)
	printPhase("Cold full analysis (process-cold, subprocess per run)", r.ColdFullAnalysis)
	if r.Baseline != nil {
		fmt.Fprintf(w, "Baseline binary: %s\n", r.Baseline.Binary)
		printPhase("Baseline cold full analysis (interleaved)", r.Baseline.ColdFullAnalysis)
	}
	printPhase("Warm full analysis (in-process loop, 1 unmeasured warmup)", r.WarmFullAnalysis)
	fmt.Fprintf(w, "Validation: accepted=%v maxCV=%.3f", r.Validation.Accepted, r.Validation.MaxCV)
	if len(r.Validation.Reasons) > 0 {
		fmt.Fprintf(w, " reasons=%s", strings.Join(r.Validation.Reasons, "; "))
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Incremental edits: supported=%v", r.Incremental.Supported)
	if r.Incremental.Reason != "" {
		fmt.Fprintf(w, " (%s)", r.Incremental.Reason)
	}
	fmt.Fprintln(w)
}
