package command

import (
	"fmt"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"os"
	"sync"
)

func ProcessFile(filePath, commandName string, debug bool) int {
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return 0
	}
	lineCount := CountLines(input)
	l := lexer.New(string(input))
	p := parser.New(l, debug)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		errCount := len(p.Errors())
		fmt.Printf("Parsing errors in %s (%d error(s)):\n", filePath, errCount)
		for _, err := range p.Errors() {
			fmt.Printf(ErrorLineFormat, err)
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
				fmt.Printf("%s: %s @ %d:%d\n", tok.Type, tok.Literal, tok.Pos.Line, tok.Pos.Column)
			}
		} else {
			cmd.Execute(nodes)
		}
	} else {
		fmt.Printf("Unknown command: %s\n", commandName)
		PrintUsage()
	}
	return lineCount
}

func ProcessFileWithErrors(filePath, commandName string, debug bool) ([]string, int) {
	input, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
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
	ExecuteCommand(commandName, nodes, input)
	return nil, lineCount
}

func ProcessMultipleFiles(files []string, commandName string, debug bool, parallelism int) (int, int) {
	totalParseErrors := 0
	totalLines := 0
	var wg sync.WaitGroup
	fileCh := make(chan string)
	linesCh := make(chan int)
	errDetailCh := make(chan ParseErrorDetail)
	linesDone := make(chan struct{})
	errorsDone := make(chan struct{})

	go CollectLines(linesCh, &totalLines, linesDone)
	go CollectErrors(errDetailCh, &totalParseErrors, errorsDone)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go Worker(fileCh, errDetailCh, linesCh, commandName, debug, &wg)
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
