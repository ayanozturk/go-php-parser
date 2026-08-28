package command

import (
	"encoding/json"
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

	if cacheDir != "" {
		cm := analyse.NewCacheManager(cacheDir)
		cachePath := cacheDir + "/go-phpcs-index.json"
		data, err := os.ReadFile(cachePath)
		if err == nil {
			var entry analyse.CacheEntry
			if json.Unmarshal(data, &entry) == nil {
				// Have cached entry: detect which files changed
				changed := cm.GetChangedFiles(&entry, checksums)
				if len(changed) == 0 {
					// No changes: skip parsing entirely, reuse cached index + analysis
					cachedIdx = analyse.NewProjectIndex()
					cachedIdx.Classes = entry.Index.Classes
					cachedIdx.Methods = entry.Index.Methods
					cachedIdx.Properties = entry.Index.Properties
					cachedIdx.Functions = entry.Index.Functions
					return analyzeWithCachedIndex(files, level, matcher, parallelism, cachedIdx, fileContents, checksums)
				}
				// Some files changed: load index for merge
				if idx, valid := cm.Load(checksums); valid {
					cachedIdx = idx
				}
			}
		}
	}

	// Parse all files (cold run or some files changed)
	return analyzeFilesWithCache(files, level, matcher, parallelism, cacheDir, cachedIdx, checksums)
}

// AnalyzeFiles parses each file once, builds one immutable project snapshot,
// and runs the registered analysis rules against that shared snapshot.
func AnalyzeFiles(files []string, level *int, matcher *overrides.Compiled, parallelism int) AnalyzeResult {
	return AnalyzeFilesIncremental(files, level, matcher, parallelism, "")
}

// analyzeWithCachedIndex runs analysis using cached index when no files changed.
// No reparsing needed; reuses cached symbols for all analysis rules.
func analyzeWithCachedIndex(files []string, level *int, matcher *overrides.Compiled, parallelism int, cachedIdx *analyse.ProjectIndex, fileContents map[string][]byte, checksums map[string]string) AnalyzeResult {
	result := AnalyzeResult{FilesDiscovered: len(files), FilesAnalyzed: len(files)}

	// Compute total lines
	for _, content := range fileContents {
		result.TotalLines += CountLines(content)
	}

	// Parse all files again (AST needed for analysis rules)
	// Note: cached index provides symbol table; we still need AST for analysis
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

	// Create snapshot with cached index (passing nil since we need fresh analysis context)
	snapshot, err := analyse.NewSemanticSnapshot(parsed, nil)
	if err != nil {
		result.ReadErrors = append(result.ReadErrors, FileReadError{File: "<project>", Message: err.Error()})
		return sortedAnalyzeResult(result)
	}

	// Run analysis using cached symbols
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

	return sortedAnalyzeResult(result)
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

	// Parse all files with AST caching
	astCacheMgr := analyse.NewASTCacheManager(cacheDir)
	jobs := make(chan string)
	parsedFiles := make(chan parsedAnalysisFile, parallelism)
	var parseWorkers sync.WaitGroup
	for i := 0; i < parallelism; i++ {
		parseWorkers.Add(1)
		go func() {
			defer parseWorkers.Done()
			for path := range jobs {
				// Try AST cache first
				checksum := checksums[path]
				if astNodes, ok := astCacheMgr.LoadAST(path, checksum); ok {
					// Cache hit: reuse AST
					content, _ := os.ReadFile(path)
					parsedFiles <- parsedAnalysisFile{
						path:        path,
						content:     content,
						nodes:       astNodes,
						lines:       CountLines(content),
						parseErrors: []string{},
					}
					continue
				}
				// Cache miss: parse normally
				paf := parseAnalysisFile(path)
				// Store AST for next run
				if paf.readError == "" && len(paf.parseErrors) == 0 {
					_ = astCacheMgr.StoreAST(paf.path, paf.nodes, checksum)
				}
				parsedFiles <- paf
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

	// Use cache if available: merge new parsed into cachedIdx instead of building from scratch
	var idx *analyse.ProjectIndex
	if cachedIdx != nil && len(parsed) > 0 {
		// Build file type contexts for parsed files
		fileContexts := make(map[string]analyse.FileTypeContext)
		for path, nodes := range parsed {
			fileContexts[path] = analyse.CollectFileTypeContext(nodes)
		}
		// Merge changed files into cached index
		cachedIdx.MergeIncremental(parsed, fileContexts)
		idx = cachedIdx
	} else {
		// No cache or cold run: build fresh index
		idx = analyse.BuildProjectIndex(parsed)
	}

	// Cache the index for next run
	if cacheDir != "" && len(parsed) > 0 {
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

// FilterAnalyzeResultToFile narrows a project-wide AnalyzeResult down to a
// single target file, e.g. for `analyze <file>`. The full project is still
// indexed for cross-file symbol resolution; only the reported diagnostics
// are scoped to the target.
func FilterAnalyzeResultToFile(result AnalyzeResult, target string) AnalyzeResult {
	return FilterAnalyzeResultToFiles(result, map[string]struct{}{target: {}})
}

// FilterAnalyzeResultToFiles narrows a project-wide AnalyzeResult down to the
// given set of reportable files. Used when config.Includes adds extra files
// to the index purely for cross-file symbol resolution: those files must not
// show up in the reported diagnostics or summary counts.
func FilterAnalyzeResultToFiles(result AnalyzeResult, keep map[string]struct{}) AnalyzeResult {
	filtered := AnalyzeResult{FilesDiscovered: len(keep)}

	issues := result.Issues[:0:0]
	for _, issue := range result.Issues {
		if _, ok := keep[issue.Filename]; ok {
			issues = append(issues, issue)
		}
	}
	filtered.Issues = issues

	parseErrors := result.ParseErrors[:0:0]
	failed := make(map[string]struct{}, len(result.ParseErrors)+len(result.ReadErrors))
	for _, pe := range result.ParseErrors {
		if _, ok := keep[pe.File]; ok {
			parseErrors = append(parseErrors, pe)
			failed[pe.File] = struct{}{}
		}
	}
	filtered.ParseErrors = parseErrors

	readErrors := result.ReadErrors[:0:0]
	for _, re := range result.ReadErrors {
		if _, ok := keep[re.File]; ok {
			readErrors = append(readErrors, re)
			failed[re.File] = struct{}{}
		}
	}
	filtered.ReadErrors = readErrors

	analyzed := 0
	totalLines := 0
	for f := range keep {
		if _, bad := failed[f]; bad {
			continue
		}
		analyzed++
		if content, err := os.ReadFile(f); err == nil {
			totalLines += CountLines(content)
		}
	}
	filtered.FilesAnalyzed = analyzed
	filtered.TotalLines = totalLines

	return filtered
}

func countParseErrors(details []ParseErrorDetail) int {
	total := 0
	for _, detail := range details {
		total += len(detail.Errors)
	}
	return total
}
