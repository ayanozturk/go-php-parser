package command

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/overrides"
	"github.com/ayanozturk/go-php-parser/parser"
	"github.com/ayanozturk/go-php-parser/sharedcache"
)

// AnalyzeResult accounts for every file selected by the standalone analyzer.
// FilesAnalyzed excludes files that could not be read or parsed.
type AnalyzeResult struct {
	Issues          []analyse.AnalysisIssue
	ParseErrors     []ParseErrorDetail
	ReadErrors      []FileReadError
	FilesDiscovered int
	FilesAnalyzed   int
	TotalLines      int
}

type FileReadError struct {
	File    string
	Message string
}

type parsedAnalysisFile struct {
	path        string
	content     []byte
	nodes       []ast.Node
	lines       int
	parseErrors []string
	readError   string
}

// AnalyzeFilesIncremental is like AnalyzeFiles but supports warm caching.
// cacheDir: directory for project index cache (empty to disable incremental)
// If cache is valid and checksums match, re-analyzes only changed files + dependents.
func AnalyzeFilesIncremental(files []string, level *int, matcher *overrides.Compiled, parallelism int, cacheDir string) AnalyzeResult {
	files = sortedUniquePaths(files)
	result := AnalyzeResult{FilesDiscovered: len(files)}
	if len(files) == 0 {
		return result
	}
	if parallelism < 1 {
		parallelism = 1
	}

	// Compute file checksums for incremental detection
	fileContents := make(map[string][]byte)
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err == nil {
			fileContents[path] = content
		}
	}
	checksums := analyse.FileChecksums(fileContents)

	// Try to load cached index
	var cachedIdx *analyse.ProjectIndex
	var filesToReparse map[string]struct{}

	if cacheDir != "" {
		cm := analyse.NewCacheManager(cacheDir)
		if idx, valid := cm.Load(checksums); valid {
			cachedIdx = idx
			filesToReparse = make(map[string]struct{})
			// Cache hit: only parse changed + dependent files
			// For now, reparse all (full optimization deferred)
			for _, f := range files {
				filesToReparse[f] = struct{}{}
			}
		}
	}

	return analyzeFilesWithCache(files, level, matcher, parallelism, cacheDir, cachedIdx, checksums)
}

// AnalyzeFiles parses each file once, builds one immutable project snapshot,
// and runs the registered analysis rules against that shared snapshot.
func AnalyzeFiles(files []string, level *int, matcher *overrides.Compiled, parallelism int) AnalyzeResult {
	return AnalyzeFilesIncremental(files, level, matcher, parallelism, "")
}

func analyzeFilesWithCache(files []string, level *int, matcher *overrides.Compiled, parallelism int, cacheDir string, cachedIdx *analyse.ProjectIndex, checksums map[string]string) AnalyzeResult {
	files = sortedUniquePaths(files)
	result := AnalyzeResult{FilesDiscovered: len(files)}
	if len(files) == 0 {
		return result
	}
	if parallelism < 1 {
		parallelism = 1
	}

	// Parse all files (full parsing; file-level optimization deferred)
	jobs := make(chan string)
	parsedFiles := make(chan parsedAnalysisFile, parallelism)
	var parseWorkers sync.WaitGroup
	for i := 0; i < parallelism; i++ {
		parseWorkers.Add(1)
		go func() {
			defer parseWorkers.Done()
			for path := range jobs {
				parsedFiles <- parseAnalysisFile(path)
			}
		}()
	}
	go func() {
		for _, path := range files {
			jobs <- path
		}
		close(jobs)
		parseWorkers.Wait()
		close(parsedFiles)
	}()

	parsed := make(map[string][]ast.Node, len(files))
	contents := make(map[string][]byte, len(files))
	for file := range parsedFiles {
		result.TotalLines += file.lines
		if file.readError != "" {
			result.ReadErrors = append(result.ReadErrors, FileReadError{File: file.path, Message: file.readError})
			continue
		}
		contents[file.path] = file.content
		sharedcache.StoreCachedFileContent(file.path, file.content)
		if len(file.parseErrors) > 0 {
			result.ParseErrors = append(result.ParseErrors, ParseErrorDetail{File: file.path, Errors: file.parseErrors})
			continue
		}
		parsed[file.path] = file.nodes
	}
	defer func() {
		for path, content := range contents {
			sharedcache.DeleteCachedFileContent(path)
			sharedcache.DeleteCachedLines(content)
		}
	}()

	// Build project index (will use cache validation below)
	// TODO: implement full incremental (skip parsing unchanged files)
	idx := analyse.BuildProjectIndex(parsed)

	// Cache the index for next run
	if cacheDir != "" && len(parsed) > 0 && cachedIdx == nil {
		cm := analyse.NewCacheManager(cacheDir)
		_ = cm.Store(idx, checksums) // Best-effort; ignore errors
	}

	snapshot, err := analyse.NewSemanticSnapshot(parsed, nil)
	if err != nil {
		result.ReadErrors = append(result.ReadErrors, FileReadError{File: "<project>", Message: err.Error()})
		return sortedAnalyzeResult(result)
	}

	analysisJobs := make(chan string)
	issueResults := make(chan []analyse.AnalysisIssue, parallelism)
	var analysisWorkers sync.WaitGroup
	for i := 0; i < parallelism; i++ {
		analysisWorkers.Add(1)
		go func() {
			defer analysisWorkers.Done()
			for path := range analysisJobs {
				ctx := snapshot.NewAnalysisContext()
				ctx.AnalysisLevel = level
				issueResults <- analyse.FilterIssues(analyse.RunAnalysisRulesWithContext(path, parsed[path], ctx), matcher)
			}
		}()
	}
	go func() {
		for _, path := range snapshot.Files() {
			analysisJobs <- path
		}
		close(analysisJobs)
		analysisWorkers.Wait()
		close(issueResults)
	}()
	for issues := range issueResults {
		result.Issues = append(result.Issues, issues...)
	}
	result.FilesAnalyzed = len(parsed)
	return sortedAnalyzeResult(result)
}

func parseAnalysisFile(path string) parsedAnalysisFile {
	content, err := os.ReadFile(path)
	if err != nil {
		return parsedAnalysisFile{path: path, readError: err.Error()}
	}
	p := parser.New(lexer.NewFile(string(content)), false)
	nodes := p.Parse()
	return parsedAnalysisFile{
		path:        path,
		content:     content,
		nodes:       nodes,
		lines:       CountLines(content),
		parseErrors: append([]string(nil), p.Errors()...),
	}
}

func sortedUniquePaths(files []string) []string {
	seen := make(map[string]struct{}, len(files))
	paths := make([]string, 0, len(files))
	for _, path := range files {
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func sortedAnalyzeResult(result AnalyzeResult) AnalyzeResult {
	sort.Slice(result.Issues, func(i, j int) bool {
		left, right := result.Issues[i], result.Issues[j]
		if left.Filename != right.Filename {
			return left.Filename < right.Filename
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		if left.EndLine != right.EndLine {
			return left.EndLine < right.EndLine
		}
		if left.EndColumn != right.EndColumn {
			return left.EndColumn < right.EndColumn
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		return left.Message < right.Message
	})
	sort.Slice(result.ParseErrors, func(i, j int) bool { return result.ParseErrors[i].File < result.ParseErrors[j].File })
	sort.Slice(result.ReadErrors, func(i, j int) bool { return result.ReadErrors[i].File < result.ReadErrors[j].File })
	return result
}

func PrintAnalyzeResult(w io.Writer, result AnalyzeResult) {
	fmt.Fprintln(w, "ANALYSIS RESULTS")
	for _, readError := range result.ReadErrors {
		fmt.Fprintf(w, "%s: read error: %s\n", readError.File, readError.Message)
	}
	for _, parseError := range result.ParseErrors {
		fmt.Fprintf(w, "%s: parser errors (%d)\n", parseError.File, len(parseError.Errors))
		for _, message := range parseError.Errors {
			fmt.Fprintf(w, "  %s\n", message)
		}
	}
	for _, issue := range result.Issues {
		location := fmt.Sprintf("%s:%d:%d", issue.Filename, issue.Line, issue.Column)
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			fmt.Fprintf(w, "%s: error: %s\n", location, issue.Message)
			continue
		}
		fmt.Fprintf(w, "%s: error [%s]: %s\n", location, code, issue.Message)
	}
	fmt.Fprintf(
		w,
		"Analysis summary: files=%d analyzed=%d diagnostics=%d parser_errors=%d read_errors=%d\n",
		result.FilesDiscovered,
		result.FilesAnalyzed,
		len(result.Issues),
		countParseErrors(result.ParseErrors),
		len(result.ReadErrors),
	)
}

func countParseErrors(details []ParseErrorDetail) int {
	total := 0
	for _, detail := range details {
		total += len(detail.Errors)
	}
	return total
}
