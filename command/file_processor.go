package command

import (
	"fmt"
	"go-phpcs/analyse"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"go-phpcs/sharedcache"
	"go-phpcs/style"
	"io"
	"os"
	"sync"
)

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

func ProcessFileWithErrors(filePath, commandName string, debug bool, w io.Writer) ([]string, int) {
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
	ExecuteCommand(commandName, nodes, input, filePath, w)
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
	return content, nil
}

func processFileForStyle(file string, rules []string, issueCh chan<- []style.StyleIssue, linesCh chan<- int, errCh chan<- int, callback func()) {
	input, err := getCachedFileContent(file)
	if err != nil {
		return
	}
	linesCh <- CountLines(input)
	lex := lexer.New(string(input))
	p := parser.New(lex, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		errCh <- len(p.Errors())
	} else {
		errCh <- 0
	}
	// Run analysis rules on the parsed AST and collect issues
	analysisIssues := analyse.RunAnalysisRules(file, nodes)
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
	Commands["style"].ExecuteWithRules(nodes, file, issueWriter, rules)
	issueCh <- fileIssues
	if callback != nil {
		callback()
	}
}

// Helper to process a batch of files in parallel
func processStyleBatch(batch []string, rules []string, parallelism int, callback func()) ([]style.StyleIssue, int, int) {
	var (
		wg      sync.WaitGroup
		fileCh  = make(chan string)
		issueCh = make(chan []style.StyleIssue, parallelism)
		linesCh = make(chan int, parallelism)
		errCh   = make(chan int, parallelism)
	)

	// Worker setup
	for j := 0; j < parallelism; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				processFileForStyle(file, rules, issueCh, linesCh, errCh, callback)
			}
		}()
	}

	// Feed files
	go func() {
		for _, file := range batch {
			fileCh <- file
		}
		close(fileCh)
	}()

	allIssues := make([]style.StyleIssue, 0, len(batch)*2)
	totalLines := 0
	totalParseErrors := 0
	for j := 0; j < len(batch); j++ {
		allIssues = append(allIssues, <-issueCh...)
		totalLines += <-linesCh
		totalParseErrors += <-errCh
	}
	close(issueCh)
	close(linesCh)
	close(errCh)
	wg.Wait()

	return allIssues, totalParseErrors, totalLines
}

func ProcessStyleFilesParallelWithCallback(files []string, rules []string, parallelism int, callback func()) ([]style.StyleIssue, int, int) {
	batchSize := 100
	nFiles := len(files)
	allIssues := make([]style.StyleIssue, 0, nFiles*2)
	totalLines := 0
	totalParseErrors := 0

	for i := 0; i < nFiles; i += batchSize {
		end := i + batchSize
		if end > nFiles {
			end = nFiles
		}
		batch := files[i:end]

		if err := PreloadFilesParallel(batch, parallelism); err != nil {
			return nil, 0, 0
		}

		batchIssues, batchParseErrors, batchLines := processStyleBatch(batch, rules, parallelism, callback)
		allIssues = append(allIssues, batchIssues...)
		totalParseErrors += batchParseErrors
		totalLines += batchLines

		for _, file := range batch {
			fileContentCache.Delete(file)
		}
	}

	return allIssues, totalParseErrors, totalLines
}

func ProcessStyleFilesParallel(files []string, rules []string, parallelism int) ([]style.StyleIssue, int, int) {
	fileContents := make(map[string][]byte, len(files))
	for _, file := range files {
		content, err := getCachedFileContent(file)
		if err == nil {
			fileContents[file] = content
		}
	}
	sharedcache.BatchTokenizeFiles(fileContents)
	if err := PreloadFilesParallel(files, 16); err != nil {
		return nil, 0, 0
	}

	var (
		wg      sync.WaitGroup
		fileCh  = make(chan string)
		issueCh = make(chan []style.StyleIssue, parallelism)
		linesCh = make(chan int, parallelism)
		errCh   = make(chan int, parallelism)
	)

	nFiles := len(files)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				processFileForStyle(file, rules, issueCh, linesCh, errCh, nil)
			}
		}()
	}

	go func() {
		for _, file := range files {
			fileCh <- file
		}
		close(fileCh)
	}()

	allIssues := make([]style.StyleIssue, 0, nFiles*2)
	totalLines := 0
	totalParseErrors := 0
	for i := 0; i < nFiles; i++ {
		allIssues = append(allIssues, <-issueCh...)
		totalLines += <-linesCh
		totalParseErrors += <-errCh
	}
	close(issueCh)
	close(linesCh)
	close(errCh)
	wg.Wait()
	return allIssues, totalParseErrors, totalLines
}

func ProcessMultipleFiles(files []string, commandName string, debug bool, parallelism int, w io.Writer) (int, int) {
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
		go Worker(fileCh, errDetailCh, linesCh, commandName, debug, &wg, w)
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

func Worker(fileCh <-chan string, errDetailCh chan<- ParseErrorDetail, linesCh chan<- int, commandName string, debug bool, wg *sync.WaitGroup, w io.Writer) {
	defer wg.Done()
	for filePath := range fileCh {
		errList, lines := ProcessFileWithErrors(filePath, commandName, debug, w)
		errDetailCh <- ParseErrorDetail{File: filePath, Errors: errList}
		linesCh <- lines
	}
}
