package printer

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"strings"
	"testing"
)

func TestPrintAST(t *testing.T) {
	var output strings.Builder
	lexer := lexer.New("<?php $name = 'John'; ?>")
	parser := parser.New(lexer, false)
	nodes := parser.Parse()
	buildStringToPrint(nodes, &output)

	expected := `Assignment(Variable($name) @ 1:8 = "John" @ 1:16) @ 1:8`

	if strings.TrimSpace(output.String()) != strings.TrimSpace(expected) {
		t.Errorf("Expected output to be %q, but got %q", strings.TrimSpace(expected), strings.TrimSpace(output.String()))
	}
}
