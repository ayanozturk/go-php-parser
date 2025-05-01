package command

import (
	"fmt"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"os"
	"sync"
	"io"
)

func ProcessFile(filePath, commandName string, debug bool, w io.Writer) int {
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(w, "Error reading file: %s\n", err)
		return 0
	}
	lineCount := CountLines(input)
	l := lexer.New(string(input))
	p := parser.New(l, debug)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		errCount := len(p.Errors())
		fmt.Fprintf(w, "Parsing errors in %s (%d error(s)):\n", filePath, errCount)
		for _, err := range p.Errors() {
			fmt.Fprintf(w, ErrorLineFormat, err)
		}
		return lineCount
	}
	if cmd, exists := Commands[commandName]; exists {
		if commandName == "tokens" {
			l := lexer.New(string(input))
			for {
				tok := l.NextToken()
				if tok.Type == "T_EOF" {
					break
				}
				fmt.Fprintf(w, "%s: %s @ %d:%d\n", tok.Type, tok.Literal, tok.Pos.Line, tok.Pos.Column)
			}
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
	input, err := os.ReadFile(filePath)
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
