package tests

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"go-phpcs/printer"
	"strings"
	"testing"
)

func TestPrintAST(t *testing.T) {
	var output strings.Builder
	lexer := lexer.New("<?php $name = 'John'; ?>")
	parser := parser.New(lexer, false)
	nodes := parser.Parse()
	printer.PrintAST(nodes, 0)

	expected := `
	Variable: $name
		String: John
	`

	if output.String() != expected {
		t.Errorf("Expected output to be %s, but got %s", expected, output.String())
	}
}
