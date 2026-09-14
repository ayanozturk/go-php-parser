// syntax-metrics scans a PHP corpus and reports syntax-kernel identity metrics.
//
// Pin-fetch (if needed):
//
//	go run ./cmd/fetch-test-projects --only symfony,wordpress-develop
//
// Full identity gate via test:
//
//	SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
//	SYNTAX_CORPUS_DIR=test_projects/wordpress-develop go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
//
// Metrics JSON report:
//
//	go run ./cmd/syntax-metrics --root test_projects/symfony --json
//	go run ./cmd/syntax-metrics --root test_projects/wordpress-develop --json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/syntax"
)

type fileResult struct {
	path           string
	bytes          int
	tokens         int
	identityPass   bool
	mismatchOffset int // -1 when pass or read error
	readError      string
}

type fileFailure struct {
	Path           string `json:"path"`
	MismatchOffset int    `json:"mismatchOffset,omitempty"`
	ReadError      string `json:"readError,omitempty"`
}

type report struct {
	GeneratedAt    string        `json:"generatedAt"`
	Root           string        `json:"root"`
	Workers        int           `json:"workers"`
	DurationMs     int64         `json:"durationMs"`
	TotalFiles     int           `json:"totalFiles"`
	PassingFiles   int           `json:"passingFiles"`
	FailingFiles   int           `json:"failingFiles"`
	IdentityPct    float64       `json:"identityPct"`
	TotalBytes     int64         `json:"totalBytes"`
	TotalTokens    int64         `json:"totalTokens"`
	TokensPerKB    float64       `json:"tokensPerKB"`
	SampleFailures []fileFailure `json:"sampleFailures,omitempty"`
}

func main() {
	root := flag.String("root", "", "root directory to scan (required)")
	workers := flag.Int("workers", runtime.NumCPU(), "number of worker goroutines")
	limit := flag.Int("limit", 0, "max files to scan (0 = all)")
	top := flag.Int("top", 5, "number of sample failing files to report")
	jsonOutput := flag.Bool("json", false, "emit JSON instead of text")
	outputPath := flag.String("output", "", "optional file to write the report to")
	flag.Parse()

	if strings.TrimSpace(*root) == "" {
		fmt.Fprintln(os.Stderr, "syntax-metrics: --root is required (e.g. --root test_projects/symfony)")
		os.Exit(2)
	}
	if *workers < 1 {
		*workers = 1
	}
	if *top < 0 {
		*top = 0
	}

	start := time.Now()
	files, err := collectPHPFiles(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "syntax-metrics: %v\n", err)
		os.Exit(1)
	}
	if *limit > 0 && len(files) > *limit {
		files = files[:*limit]
	}

	results := scanFiles(files, *workers)
	rep := buildReport(*root, *workers, *top, start, results)

	out := io.Writer(os.Stdout)
	if *outputPath != "" {
		file, createErr := os.Create(*outputPath)
		if createErr != nil {
			fmt.Fprintf(os.Stderr, "syntax-metrics: %v\n", createErr)
			os.Exit(1)
		}
		defer file.Close()
		out = file
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintf(os.Stderr, "syntax-metrics: %v\n", err)
			os.Exit(1)
		}
	} else {
		printTextReport(out, rep)
	}

	if rep.FailingFiles > 0 {
		os.Exit(1)
	}
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

func scanFiles(files []string, workers int) []fileResult {
	jobs := make(chan string)
	results := make(chan fileResult, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				results <- checkFile(path)
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

func checkFile(path string) fileResult {
	content, err := os.ReadFile(path)
	if err != nil {
		return fileResult{
			path:           path,
			identityPass:   false,
			mismatchOffset: -1,
			readError:      err.Error(),
		}
	}

	toks := lexer.LexAll(content)
	res := syntax.Parse(content)
	got := syntax.Print(res.File.Root)
	want := string(content)
	pass := got == want
	offset := -1
	if !pass {
		offset = firstMismatchOffset(want, got)
	}

	return fileResult{
		path:           path,
		bytes:          len(content),
		tokens:         len(toks),
		identityPass:   pass,
		mismatchOffset: offset,
	}
}

func firstMismatchOffset(want, got string) int {
	n := len(want)
	if len(got) < n {
		n = len(got)
	}
	for i := 0; i < n; i++ {
		if want[i] != got[i] {
			return i
		}
	}
	if len(want) != len(got) {
		return n
	}
	return -1
}

func buildReport(root string, workers, top int, start time.Time, results []fileResult) report {
	rep := report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Root:        root,
		Workers:     workers,
		DurationMs:  time.Since(start).Milliseconds(),
		TotalFiles:  len(results),
	}

	var failures []fileFailure
	for _, result := range results {
		rep.TotalBytes += int64(result.bytes)
		rep.TotalTokens += int64(result.tokens)
		if result.identityPass {
			rep.PassingFiles++
			continue
		}
		rep.FailingFiles++
		failures = append(failures, fileFailure{
			Path:           result.path,
			MismatchOffset: result.mismatchOffset,
			ReadError:      result.readError,
		})
	}

	rep.IdentityPct = pct(rep.PassingFiles, rep.TotalFiles)
	if rep.TotalBytes > 0 {
		rep.TokensPerKB = float64(rep.TotalTokens) / (float64(rep.TotalBytes) / 1024.0)
	}

	if top > 0 && len(failures) > top {
		failures = failures[:top]
	}
	rep.SampleFailures = failures
	return rep
}

func pct(passing, total int) float64 {
	if total == 0 {
		return 100
	}
	return float64(passing) * 100 / float64(total)
}

func printTextReport(w io.Writer, rep report) {
	fmt.Fprintf(w, "Syntax Identity Metrics\n")
	fmt.Fprintf(w, "Root: %s\n", rep.Root)
	fmt.Fprintf(w, "Generated: %s\n", rep.GeneratedAt)
	fmt.Fprintf(w, "Scanned %d PHP files in %dms using %d workers\n", rep.TotalFiles, rep.DurationMs, rep.Workers)
	fmt.Fprintf(w, "Identity: %.2f%% passing (%d/%d), %d failing\n",
		rep.IdentityPct, rep.PassingFiles, rep.TotalFiles, rep.FailingFiles)
	fmt.Fprintf(w, "Totals: %d bytes, %d tokens, %.1f tokens/KB\n\n",
		rep.TotalBytes, rep.TotalTokens, rep.TokensPerKB)

	if len(rep.SampleFailures) == 0 {
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FAILING_FILE\tMISMATCH_OFFSET\tREAD_ERROR")
	for _, failure := range rep.SampleFailures {
		offset := ""
		if failure.MismatchOffset >= 0 {
			offset = fmt.Sprintf("%d", failure.MismatchOffset)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", failure.Path, offset, failure.ReadError)
	}
	_ = tw.Flush()
}
