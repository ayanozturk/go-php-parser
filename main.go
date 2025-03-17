package main

import (
	"fmt"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"go-phpcs/style"
	"os"
)

func main() {
	data, _ := os.ReadFile("examples/test.php")
	l := lexer.New(string(data))
	p := parser.New(l)
	nodes := p.Parse()
	style.Check(nodes)
	fmt.Println("Finished style check")
}
