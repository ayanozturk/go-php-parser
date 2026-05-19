package command

import (
	"fmt"
	"go-phpcs/analyse"
	"go-phpcs/lexer"
	"go-phpcs/overrides"
	"go-phpcs/parser"
	"go-phpcs/sharedcache"
	"go-phpcs/style"
	"io"
	"os"
	"sync"
	"time"
)

// fileParseTimeout is the maximum time allowed to parse + analyse a single file.
// Files that exceed this limit are skipped with a stderr warning; this prevents
// a buggy lexer/parser edge-case from hanging the entire scan.
const fileParseTimeout = 10 * time.Second

// fileResult carries all per-file outputs through a single channel to avoid
// the ordering deadlock that occurs when issueCh/linesCh/errCh are separate.
type fileResult struct {
	issues []style.StyleIssue
	lines  int
	errors int
}

func handleParsingErrors(p *parser.Parser, filePath string, w io.Writer, lineCount int) int {
	errCount := len(p.Errors())
	fmt.Fprintf(w, "Parsing errors in %s (%d error(s)):\n", filePath, errCount)
	for _, err := range p.Errors() {
		fmt.Fprintf(w, ErrorLineFormat, err)
	}
	return lineCount
}

func handleTokensCommand(input []byte, w io.Writer) {
	l := lexer.New(string(input))
	for {
		tok := l.NextToken()
		if tok.Type == "T_EOF" {
			break
		}
		fmt.Fprintf(w, "%s: %s @ %d:%d\n", tok.Type, tok.Literal, tok.Pos.Line, tok.Pos.Column)
	}
}

func ProcessFile(filePath, commandName string, debug bool, w io.Writer) int {
	input, err := getCachedFileContent(filePath)
	if err != nil {
		fmt.Fprintf(w, "Error reading file: %s\n", err)
		return 0
	}
	lineCount := CountLines(input)
	l := lexer.New(string(input))
	p := parser.New(l, debug)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		return handleParsingErrors(p, filePath, w, lineCount)
	}
	if cmd, exists := Commands[commandName]; exists {
		if commandName == "tokens" {
			handleTokensCommand(input, w)
		} else {
			cmd.Execute(nodes, filePath, w)
		}
	} else {
		fmt.Fprintf(w, "Unknown command: %s\n", commandName)
		PrintUsage()
	}
	return lineCount
}

func ProcessFileWithErrors(filePath, commandName string, debug bool, rules []string, matcher *overrides.Compiled, w io.Writer) ([]string, int) {
	input, err := getCachedFileContent(filePath)
	if err != nil {
		fmt.Fprintf(w, "Error reading file: %s\n", err)
		return nil, 0
	}
	lineCount := CountLines(input)
	l := lexer.New(string(input))
	p := parser.New(l, debug)
	nodes := p.Parse()
	errList := p.Errors()
	if len(errList) > 0 {
		return errList, lineCount
	}
	if commandName == "style" {
		Commands["style"].ExecuteWithRules(nodes, filePath, w, rules, matcher)
	} else {
		ExecuteCommand(commandName, nodes, input, filePath, w)
	}
	return nil, lineCount
}

// ProcessStyleFilesParallelWithCallback scans files in parallel, parses once per file, applies all rules, collects issues, and calls the callback after each file.

var fileContentCache sync.Map

// PreloadFilesParallel reads all files in parallel and stores their contents in fileContentCache.
func PreloadFilesParallel(files []string, parallelism int) error {
	var wg sync.WaitGroup
	fileCh := make(chan string)
	errCh := make(chan error, parallelism)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				if _, ok := fileContentCache.Load(file); ok {
					continue
				}
				content, err := os.ReadFile(file)
				if err != nil {
					errCh <- err
					continue
				}
				fileContentCache.Store(file, content)
				sharedcache.StoreCachedFileContent(file, content)
			}
		}()
	}

	go func() {
		for _, file := range files {
			fileCh <- file
		}
		close(fileCh)
	}()

	wg.Wait()
	close(errCh)
	if len(errCh) > 0 {
		return <-errCh
	}
	return nil
}

func getCachedFileContent(filename string) ([]byte, error) {
	if val, ok := fileContentCache.Load(filename); ok {
		return val.([]byte), nil
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	fileContentCache.Store(filename, content)
	sharedcache.StoreCachedFileContent(filename, content)
	return content, nil
}

func processFileForStyle(file string, rules []string, matcher *overrides.Compiled, resultCh chan<- fileResult, callback func()) {
	input, err := getCachedFileContent(file)
	if err != nil {
		// Always send a result so the collector loop never hangs.
		resultCh <- fileResult{}
		if callback != nil {
			callback()
		}
		return
	}
	lines := CountLines(input)
	lex := lexer.New(string(input))
	p := parser.New(lex, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		resultCh <- fileResult{lines: lines, errors: len(p.Errors())}
		if callback != nil {
			callback()
		}
		return
	}
	// Run analysis rules on the parsed AST and collect issues
	analysisIssues := analyse.FilterIssues(analyse.RunAnalysisRules(file, nodes), matcher)
	var fileIssues []style.StyleIssue
	for _, iss := range analysisIssues {
		// Convert AnalysisIssue to StyleIssue for unified reporting
		fileIssues = append(fileIssues, style.StyleIssue{
			Filename: iss.Filename,
			Line:     iss.Line,
			Column:   iss.Column,
			Type:     style.Error,
			Fixable:  false,
			Message:  iss.Message,
			Code:     iss.Code,
		})
	}
	issueWriter := &style.IssueCollector{Issues: &fileIssues}
	Commands["style"].ExecuteWithRules(nodes, file, issueWriter, rules, matcher)
	resultCh <- fileResult{issues: fileIssues, lines: lines, errors: 0}
	if callback != nil {
		callback()
	}
}

// Helper to process a batch of files in parallel.
// Uses a single resultCh (unified struct) to avoid the ordering deadlock that
// occurred with separate issueCh/linesCh/errCh channels.
func processStyleBatch(batch []string, rules []string, matcher *overrides.Compiled, parallelism int, callback func()) ([]style.StyleIssue, int, int) {
	n := len(batch)
	fileCh := make(chan string, parallelism)
	resultCh := make(chan fileResult, parallelism)

	var wg sync.WaitGroup
	for j := 0; j < parallelism; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				processFileForStyle(file, rules, matcher, resultCh, callback)
			}
		}()
	}

	go func() {
		for _, file := range batch {
			fileCh <- file
		}
		close(fileCh)
	}()

	allIssues := make([]style.StyleIssue, 0, n*2)
	totalLines := 0
	totalParseErrors := 0
	for j := 0; j < n; j++ {
		res := <-resultCh
		allIssues = append(allIssues, res.issues...)
		totalLines += res.lines
		totalParseErrors += res.errors
	}
	wg.Wait()

	return allIssues, totalParseErrors, totalLines
}

// ProcessStyleFilesParallelWithCallback scans all files in a streaming pipeline:
// - ioWorkers goroutines read files from disk (I/O bound, more workers than CPUs)
// - parallelism goroutines parse + analyse (CPU bound)
// I/O and CPU overlap fully; no preload→process serialization.
func ProcessStyleFilesParallelWithCallback(files []string, rules []string, matcher *overrides.Compiled, parallelism int, callback func()) ([]style.StyleIssue, int, int) {
	nFiles := len(files)
	if nFiles == 0 {
		return nil, 0, 0
	}

	// Use more goroutines for I/O than CPU — high-latency volumes benefit greatly.
	ioWorkers := parallelism * 4
	if ioWorkers < 16 {
		ioWorkers = 16
	}

	type readResult struct {
		path    string
		content []byte
	}

	pathCh := make(chan string, ioWorkers*2)
	contentCh := make(chan readResult, parallelism*4)
	resultCh := make(chan fileResult, parallelism)

	// Feed paths
	go func() {
		for _, f := range files {
			pathCh <- f
		}
		close(pathCh)
	}()

	// I/O workers: read files concurrently
	var ioWg sync.WaitGroup
	for i := 0; i < ioWorkers; i++ {
		ioWg.Add(1)
		go func() {
			defer ioWg.Done()
			for path := range pathCh {
				content, err := os.ReadFile(path)
				if err != nil {
					content = nil
				}
				contentCh <- readResult{path: path, content: content}
			}
		}()
	}
	go func() {
		ioWg.Wait()
		close(contentCh)
	}()

	// CPU workers: parse + analyse
	var cpuWg sync.WaitGroup
	for i := 0; i < parallelism; i++ {
		cpuWg.Add(1)
		go func() {
			defer cpuWg.Done()
			for rr := range contentCh {
				if rr.content == nil {
					resultCh <- fileResult{}
					if callback != nil {
						callback()
					}
					continue
				}
				// Store in shared cache so style rules can access raw content.
				sharedcache.StoreCachedFileContent(rr.path, rr.content)

				lines := CountLines(rr.content)

				// Run parse+analyse in an inner goroutine with a hard timeout.
				// A lexer/parser edge-case on a malformed file can spin indefinitely;
				// the timeout guarantees the cpu worker is never permanently blocked.
				type innerResult struct {
					issues []style.StyleIssue
					errors int
				}
				done := make(chan innerResult, 1)
				go func(path string, content []byte) {
					lex := lexer.New(string(content))
					p := parser.New(lex, false)
					nodes := p.Parse()
					if len(p.Errors()) > 0 {
						done <- innerResult{errors: len(p.Errors())}
						return
					}
					analysisIssues := analyse.FilterIssues(analyse.RunAnalysisRules(path, nodes), matcher)
					var fileIssues []style.StyleIssue
					for _, iss := range analysisIssues {
						fileIssues = append(fileIssues, style.StyleIssue{
							Filename: iss.Filename,
							Line:     iss.Line,
							Column:   iss.Column,
							Type:     style.Error,
							Fixable:  false,
							Message:  iss.Message,
							Code:     iss.Code,
						})
					}
					issueWriter := &style.IssueCollector{Issues: &fileIssues}
					Commands["style"].ExecuteWithRules(nodes, path, issueWriter, rules, matcher)
					done <- innerResult{issues: fileIssues}
				}(rr.path, rr.content)

				var res innerResult
				select {
				case res = <-done:
				case <-time.After(fileParseTimeout):
					fmt.Fprintf(os.Stderr, "\n[warn] parse timeout (%s): skipping %s\n", fileParseTimeout, rr.path)
					// The inner goroutine is leaked but will eventually exit (or the
					// process ends). The timeout ensures the scan completes.
				}
				sharedcache.DeleteCachedFileContent(rr.path)
				resultCh <- fileResult{issues: res.issues, lines: lines, errors: res.errors}
				if callback != nil {
					callback()
				}
			}
		}()
	}
	go func() {
		cpuWg.Wait()
		close(resultCh)
	}()

	// Collect results
	allIssues := make([]style.StyleIssue, 0, nFiles*2)
	totalLines := 0
	totalParseErrors := 0
	for res := range resultCh {
		allIssues = append(allIssues, res.issues...)
		totalLines += res.lines
		totalParseErrors += res.errors
	}

	return allIssues, totalParseErrors, totalLines
}

func ProcessStyleFilesParallel(files []string, rules []string, matcher *overrides.Compiled, parallelism int) ([]style.StyleIssue, int, int) {
	return ProcessStyleFilesParallelWithCallback(files, rules, matcher, parallelism, nil)
}

func ProcessMultipleFiles(files []string, commandName string, debug bool, parallelism int, rules []string, matcher *overrides.Compiled, w io.Writer) (int, int) {
	totalParseErrors := 0
	totalLines := 0
	var wg sync.WaitGroup
	fileCh := make(chan string)
	linesCh := make(chan int)
	errDetailCh := make(chan ParseErrorDetail)
	linesDone := make(chan struct{})
	errorsDone := make(chan struct{})

	go CollectLines(linesCh, &totalLines, linesDone)
	go CollectErrors(errDetailCh, &totalParseErrors, errorsDone, w)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go Worker(fileCh, errDetailCh, linesCh, commandName, debug, rules, matcher, &wg, w)
	}

	for _, filePath := range files {
		fileCh <- filePath
	}
	close(fileCh)
	wg.Wait()
	close(linesCh)
	close(errDetailCh)
	<-linesDone
	<-errorsDone

	return totalParseErrors, totalLines
}

func CollectErrors(errDetailCh <-chan ParseErrorDetail, totalParseErrors *int, done chan<- struct{}, w io.Writer) {
	for errDetail := range errDetailCh {
		if len(errDetail.Errors) > 0 {
			*totalParseErrors += len(errDetail.Errors)
			fmt.Fprintf(w, "\nParsing errors in %s (%d error(s)):\n", errDetail.File, len(errDetail.Errors))
			for _, err := range errDetail.Errors {
				fmt.Fprintf(w, ErrorLineFormat, err)
			}
		}
	}
	done <- struct{}{}
}

func Worker(fileCh <-chan string, errDetailCh chan<- ParseErrorDetail, linesCh chan<- int, commandName string, debug bool, rules []string, matcher *overrides.Compiled, wg *sync.WaitGroup, w io.Writer) {
	defer wg.Done()
	for filePath := range fileCh {
		errList, lines := ProcessFileWithErrors(filePath, commandName, debug, rules, matcher, w)
		errDetailCh <- ParseErrorDetail{File: filePath, Errors: errList}
		linesCh <- lines
	}
}
