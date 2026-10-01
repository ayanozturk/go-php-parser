// analysis-corpus-snapshot is the issue-set parity guardrail for the
// production analysis registry and its CST-direct migrations (see AGENTS.md).
//
// It runs the full analysis-rule registry (via analyse.RunAnalysisRulesWithContext,
// exactly as command/analyze.go does for real projects) over every PHP file
// under --root and records the resulting issue set per file. The harness
// indexes the corpus once, then parses and analyzes one target file at a time;
// each registry run receives the same lowered AST and caller-owned ParseResult
// so CST rules reuse that file's parse. Compare with --baseline after an
// analysis refactor to prove the exact same issues are produced, file for
// file, code/line/column/message.
//
// Capture a baseline:
//
//	go run ./cmd/analysis-corpus-snapshot --root test_projects/wordpress-develop --output /tmp/baseline.json
//
// Compare after a change:
//
//	go run ./cmd/analysis-corpus-snapshot --root test_projects/wordpress-develop --baseline /tmp/baseline.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

const reportSchemaVersion = 1

type snapshot struct {
	SchemaVersion int                 `json:"schemaVersion"`
	GeneratedAt   string              `json:"generatedAt"`
	Root          string              `json:"root"`
	Level         int                 `json:"level,omitempty"`
	TotalFiles    int                 `json:"totalFiles"`
	ParseErrors   []string            `json:"parseErrors,omitempty"`
	Issues        map[string][]string `json:"issues"`
}

func main() {
	root := flag.String("root", "", "root directory to scan (required)")
	workers := flag.Int("workers", runtime.NumCPU(), "number of worker goroutines")
	limit := flag.Int("limit", 0, "max files to scan (0 = all)")
	analysisOffset := flag.Int("analysis-offset", 0, "number of indexed files to skip before analysis (0 = first file)")
	analysisLimit := flag.Int("analysis-limit", 0, "max indexed files to analyze (0 = all)")
	level := flag.Int("level", -1, "analysis level to run (-1 = every registered rule, matching a nil AnalysisLevel)")
	output := flag.String("output", "", "file to write the JSON snapshot to (required unless --baseline is set)")
	baseline := flag.String("baseline", "", "previously captured snapshot JSON to diff the current run against")
	memProfile := flag.String("memprofile", "", "optional Go heap profile to write after building the project index")
	memProfileDuringAnalysis := flag.String("memprofile-during-analysis", "", "optional Go heap profile path for a delayed capture during target analysis")
	memProfileAnalysisDelay := flag.Duration("memprofile-analysis-delay", 0, "delay before --memprofile-during-analysis is captured (for example 15s)")
	memProfileAfter := flag.String("memprofile-after", "", "optional Go heap profile to write after corpus analysis")
	maxDiffExamples := flag.Int("max-diff-examples", 20, "max per-file diff examples to print")
	flag.Parse()

	if strings.TrimSpace(*root) == "" {
		fmt.Fprintln(os.Stderr, "analysis-corpus-snapshot: --root is required (e.g. --root test_projects/wordpress-develop)")
		os.Exit(2)
	}
	if *output == "" && *baseline == "" {
		fmt.Fprintln(os.Stderr, "analysis-corpus-snapshot: one of --output or --baseline is required")
		os.Exit(2)
	}
	if *workers < 1 {
		*workers = 1
	}

	files, err := collectPHPFiles(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "analysis-corpus-snapshot: %v\n", err)
		os.Exit(1)
	}
	if *limit > 0 && len(files) > *limit {
		files = files[:*limit]
	}

	current, err := buildSnapshot(*root, files, *workers, *level, *analysisOffset, *analysisLimit, *memProfile, *memProfileDuringAnalysis, *memProfileAnalysisDelay, *memProfileAfter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "analysis-corpus-snapshot: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		if err := writeSnapshot(*output, current); err != nil {
			fmt.Fprintf(os.Stderr, "analysis-corpus-snapshot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote snapshot for %d files (%d parse errors) to %s\n", current.TotalFiles, len(current.ParseErrors), *output)
	}

	if *baseline == "" {
		return
	}
	base, err := readSnapshot(*baseline)
	if err != nil {
		fmt.Fprintf(os.Stderr, "analysis-corpus-snapshot: %v\n", err)
		os.Exit(1)
	}
	mismatches := diffSnapshots(base, current, *maxDiffExamples)
	if mismatches > 0 {
		fmt.Printf("%d file(s) differ from baseline %s\n", mismatches, *baseline)
		os.Exit(1)
	}
	fmt.Printf("matches baseline %s (%d files)\n", *baseline, current.TotalFiles)
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

func buildSnapshot(root string, files []string, workers, level, analysisOffset, analysisLimit int, memProfile, memProfileDuringAnalysis string, memProfileAnalysisDelay time.Duration, memProfileAfter string) (*snapshot, error) {
	var parseErrors []string
	var mu sync.Mutex
	projectFiles := make([]string, 0, len(files))
	project, err := analyse.BuildProjectIndexFromFiles(files, func(path string) ([]ast.Node, error) {
		content, err := os.ReadFile(path)
		if err != nil {
			mu.Lock()
			parseErrors = append(parseErrors, fmt.Sprintf("%s: read: %v", path, err))
			mu.Unlock()
			return nil, nil
		}
		nodes, res := syntax.ParseAndLower(content)
		if res == nil || len(res.Diagnostics) > 0 {
			diagCount := 0
			if res != nil {
				diagCount = len(res.Diagnostics)
			}
			mu.Lock()
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %d parse diagnostic(s)", path, diagCount))
			mu.Unlock()
			return nil, nil
		}
		projectFiles = append(projectFiles, path)
		return nodes, nil
	})
	if err != nil {
		return nil, fmt.Errorf("build project index: %w", err)
	}
	runtime.GC()
	if memProfile != "" {
		if err := writeHeapProfile(memProfile); err != nil {
			return nil, err
		}
	}
	if analysisOffset < 0 {
		analysisOffset = 0
	}
	if analysisOffset > len(projectFiles) {
		analysisOffset = len(projectFiles)
	}
	targetFiles := projectFiles[analysisOffset:]
	if analysisLimit > 0 && len(targetFiles) > analysisLimit {
		targetFiles = targetFiles[:analysisLimit]
	}
	snap, err := analyse.NewSemanticSnapshotWithIndexOnly(project, targetFiles, nil)
	if err != nil {
		return nil, fmt.Errorf("build semantic snapshot: %w", err)
	}
	targets := snap.Files()
	var analysisProfileTimer *time.Timer
	var analysisProfileDone chan struct{}
	if memProfileDuringAnalysis != "" && memProfileAnalysisDelay > 0 {
		analysisProfileDone = make(chan struct{})
		analysisProfileTimer = time.AfterFunc(memProfileAnalysisDelay, func() {
			defer close(analysisProfileDone)
			if err := writeHeapProfile(memProfileDuringAnalysis); err != nil {
				fmt.Fprintf(os.Stderr, "analysis-corpus-snapshot: writing delayed heap profile: %v\n", err)
				return
			}
			fmt.Fprintf(os.Stderr, "analysis-corpus-snapshot: wrote delayed heap profile to %s\n", memProfileDuringAnalysis)
		})
	}
	results := make(map[string][]string, len(targets))
	var resultsMu sync.Mutex
	targetJobs := make(chan string)
	var runWg sync.WaitGroup
	runWg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer runWg.Done()
			for path := range targetJobs {
				content, err := os.ReadFile(path)
				if err != nil {
					mu.Lock()
					parseErrors = append(parseErrors, fmt.Sprintf("%s: read on analysis pass: %v", path, err))
					mu.Unlock()
					continue
				}
				nodes, res := syntax.ParseAndLower(content)
				if res == nil || len(res.Diagnostics) > 0 {
					diagCount := 0
					if res != nil {
						diagCount = len(res.Diagnostics)
					}
					mu.Lock()
					parseErrors = append(parseErrors, fmt.Sprintf("%s: %d parse diagnostic(s) on analysis pass", path, diagCount))
					mu.Unlock()
					continue
				}
				ctx := snap.AnalysisContextForFile(path, nodes)
				if level >= 0 {
					l := level
					ctx.AnalysisLevel = &l
				}
				ctx.Content = content
				ctx.Parsed = res
				issues := analyse.RunAnalysisRulesWithContext(path, nodes, ctx)
				codes := make([]string, len(issues))
				for i, issue := range issues {
					codes[i] = fmt.Sprintf("%s|%d|%d|%d|%d|%s", issue.Code, issue.Line, issue.Column, issue.EndLine, issue.EndColumn, issue.Message)
				}
				sort.Strings(codes)
				resultsMu.Lock()
				results[path] = codes
				resultsMu.Unlock()
			}
		}()
	}
	for _, path := range targets {
		targetJobs <- path
	}
	close(targetJobs)
	runWg.Wait()
	if analysisProfileTimer != nil && !analysisProfileTimer.Stop() {
		<-analysisProfileDone
	}
	sort.Strings(parseErrors)
	if memProfileAfter != "" {
		if err := writeHeapProfile(memProfileAfter); err != nil {
			return nil, err
		}
	}

	return &snapshot{
		SchemaVersion: reportSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Root:          root,
		Level:         level,
		TotalFiles:    len(targets),
		ParseErrors:   parseErrors,
		Issues:        results,
	}, nil
}

func writeHeapProfile(path string) error {
	runtime.GC()
	profile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create heap profile %s: %w", path, err)
	}
	if err := pprof.WriteHeapProfile(profile); err != nil {
		_ = profile.Close()
		return fmt.Errorf("write heap profile %s: %w", path, err)
	}
	if err := profile.Close(); err != nil {
		return fmt.Errorf("close heap profile %s: %w", path, err)
	}
	return nil
}

func writeSnapshot(path string, snap *snapshot) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(snap)
}

func readSnapshot(path string) (*snapshot, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		return nil, fmt.Errorf("decode snapshot %s: %w", path, err)
	}
	if snap.SchemaVersion != reportSchemaVersion {
		return nil, fmt.Errorf("snapshot %s has unsupported schema version %d", path, snap.SchemaVersion)
	}
	return &snap, nil
}

func diffSnapshots(base, current *snapshot, maxExamples int) int {
	mismatches := 0
	examples := 0
	seen := make(map[string]bool, len(base.Issues)+len(current.Issues))
	paths := make([]string, 0, len(base.Issues)+len(current.Issues))
	for p := range base.Issues {
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	for p := range current.Issues {
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)

	for _, path := range paths {
		baseIssues := base.Issues[path]
		curIssues := current.Issues[path]
		if equalStrings(baseIssues, curIssues) {
			continue
		}
		mismatches++
		if examples >= maxExamples {
			continue
		}
		examples++
		fmt.Printf("--- %s\n", path)
		for _, removed := range setDiff(baseIssues, curIssues) {
			fmt.Printf("  -%s\n", removed)
		}
		for _, added := range setDiff(curIssues, baseIssues) {
			fmt.Printf("  +%s\n", added)
		}
	}
	return mismatches
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// setDiff returns the sorted elements of a not present in b.
func setDiff(a, b []string) []string {
	inB := make(map[string]bool, len(b))
	for _, v := range b {
		inB[v] = true
	}
	var out []string
	for _, v := range a {
		if !inB[v] {
			out = append(out, v)
		}
	}
	return out
}
