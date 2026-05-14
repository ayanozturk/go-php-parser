package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

type fileResult struct {
	path       string
	project    string
	errorCount int
	firstError string
	readError  string
}

type fileFailure struct {
	Path       string `json:"path"`
	ErrorCount int    `json:"errorCount"`
	FirstError string `json:"firstError"`
}

type projectReport struct {
	Project          string        `json:"project"`
	TotalFiles       int           `json:"totalFiles"`
	PassingFiles     int           `json:"passingFiles"`
	FailingFiles     int           `json:"failingFiles"`
	CompatibilityPct float64       `json:"compatibilityPct"`
	TotalParseErrors int           `json:"totalParseErrors"`
	SampleFailures   []fileFailure `json:"sampleFailures,omitempty"`
}

type report struct {
	GeneratedAt      string          `json:"generatedAt"`
	Root             string          `json:"root"`
	Workers          int             `json:"workers"`
	DurationMs       int64           `json:"durationMs"`
	TotalFiles       int             `json:"totalFiles"`
	PassingFiles     int             `json:"passingFiles"`
	FailingFiles     int             `json:"failingFiles"`
	CompatibilityPct float64         `json:"compatibilityPct"`
	TotalParseErrors int             `json:"totalParseErrors"`
	Projects         []projectReport `json:"projects"`
}

func main() {
	root := flag.String("root", "test_projects", "root directory to scan")
	workers := flag.Int("workers", runtime.NumCPU(), "number of worker goroutines")
	top := flag.Int("top", 3, "number of sample failing files to report per project")
	jsonOutput := flag.Bool("json", false, "emit JSON instead of text")
	outputPath := flag.String("output", "", "optional file to write the report to")
	flag.Parse()

	if *workers < 1 {
		*workers = 1
	}
	if *top < 0 {
		*top = 0
	}

	start := time.Now()
	files, err := collectPHPFiles(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compat-metrics: %v\n", err)
		os.Exit(1)
	}

	results := scanFiles(files, *root, *workers)
	report := buildReport(*root, *workers, *top, start, results)

	out := io.Writer(os.Stdout)
	if *outputPath != "" {
		file, createErr := os.Create(*outputPath)
		if createErr != nil {
			fmt.Fprintf(os.Stderr, "compat-metrics: %v\n", createErr)
			os.Exit(1)
		}
		defer file.Close()
		out = file
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "compat-metrics: %v\n", err)
			os.Exit(1)
		}
		return
	}

	printTextReport(out, report)
}

func collectPHPFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".php") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func scanFiles(files []string, root string, workers int) []fileResult {
	jobs := make(chan string)
	results := make(chan fileResult, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				results <- parseFile(path, root)
			}
		}()
	}

	go func() {
		for _, path := range files {
			jobs <- path
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	all := make([]fileResult, 0, len(files))
	for result := range results {
		all = append(all, result)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].path < all[j].path })
	return all
}

func parseFile(path, root string) fileResult {
	content, err := os.ReadFile(path)
	if err != nil {
		return fileResult{
			path:       path,
			project:    projectName(root, path),
			errorCount: 1,
			firstError: err.Error(),
			readError:  err.Error(),
		}
	}

	l := lexer.New(string(content))
	p := parser.New(l, false)
	_ = p.Parse()
	errs := p.Errors()

	result := fileResult{
		path:    path,
		project: projectName(root, path),
	}
	if len(errs) > 0 {
		result.errorCount = len(errs)
		result.firstError = errs[0]
	}
	return result
}

func projectName(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(filepath.Dir(path))
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 || parts[0] == "." || parts[0] == "" {
		return filepath.Base(root)
	}
	return parts[0]
}

func buildReport(root string, workers, top int, start time.Time, results []fileResult) report {
	projects := map[string]*projectReport{}
	failures := map[string][]fileFailure{}
	compat := report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Root:        root,
		Workers:     workers,
		DurationMs:  time.Since(start).Milliseconds(),
	}

	for _, result := range results {
		compat.TotalFiles++
		project := projects[result.project]
		if project == nil {
			project = &projectReport{Project: result.project}
			projects[result.project] = project
		}
		project.TotalFiles++

		if result.errorCount == 0 {
			compat.PassingFiles++
			project.PassingFiles++
			continue
		}

		compat.FailingFiles++
		compat.TotalParseErrors += result.errorCount
		project.FailingFiles++
		project.TotalParseErrors += result.errorCount
		failures[result.project] = append(failures[result.project], fileFailure{
			Path:       result.path,
			ErrorCount: result.errorCount,
			FirstError: result.firstError,
		})
	}

	compat.CompatibilityPct = pct(compat.PassingFiles, compat.TotalFiles)

	projectNames := make([]string, 0, len(projects))
	for name := range projects {
		projectNames = append(projectNames, name)
	}
	sort.Strings(projectNames)
	compat.Projects = make([]projectReport, 0, len(projectNames))
	for _, name := range projectNames {
		project := projects[name]
		project.CompatibilityPct = pct(project.PassingFiles, project.TotalFiles)
		if top > 0 {
			project.SampleFailures = failures[name]
			if len(project.SampleFailures) > top {
				project.SampleFailures = project.SampleFailures[:top]
			}
		}
		compat.Projects = append(compat.Projects, *project)
	}

	return compat
}

func pct(passing, total int) float64 {
	if total == 0 {
		return 100
	}
	return float64(passing) * 100 / float64(total)
}

func printTextReport(w io.Writer, report report) {
	fmt.Fprintf(w, "Compatibility Metrics\n")
	fmt.Fprintf(w, "Root: %s\n", report.Root)
	fmt.Fprintf(w, "Generated: %s\n", report.GeneratedAt)
	fmt.Fprintf(w, "Scanned %d PHP files in %dms using %d workers\n", report.TotalFiles, report.DurationMs, report.Workers)
	fmt.Fprintf(w, "Overall: %.2f%% compatible (%d/%d passing), %d failing files, %d parse errors\n\n",
		report.CompatibilityPct,
		report.PassingFiles,
		report.TotalFiles,
		report.FailingFiles,
		report.TotalParseErrors,
	)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PROJECT\tTOTAL\tPASS\tFAIL\tCOMPAT\tPARSE_ERRORS")
	for _, project := range report.Projects {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%.2f%%\t%d\n",
			project.Project,
			project.TotalFiles,
			project.PassingFiles,
			project.FailingFiles,
			project.CompatibilityPct,
			project.TotalParseErrors,
		)
	}
	_ = tw.Flush()

	for _, project := range report.Projects {
		if len(project.SampleFailures) == 0 {
			continue
		}
		fmt.Fprintf(w, "\nFirst failing files for %s:\n", project.Project)
		for _, failure := range project.SampleFailures {
			fmt.Fprintf(w, "  %s\n", failure.Path)
			fmt.Fprintf(w, "    %s\n", failure.FirstError)
		}
	}
}
