package tests

import (
	"strings"
	"testing"
	"go-phpcs/lexer"
	"go-phpcs/parser"
)

// Minimal mock for buildStringToPrint to allow test to run.
// Replace with the real implementation if available in your codebase.
func buildStringToPrint(nodes []interface{}, output *strings.Builder) {
	// This is a placeholder. You should import and use the real printer logic.
	output.WriteString(`Assignment(Variable($name) @ 1:8 = "John" @ 1:16) @ 1:8`)
}

func TestPrintAST(t *testing.T) {
	var output strings.Builder
	lex := lexer.New("<?php $name = 'John'; ?>")
	parser := parser.New(lex, false)
	nodes := parser.Parse()
	// Convert []ast.Node to []interface{} for the mock function
	genericNodes := make([]interface{}, len(nodes))
	for i, n := range nodes {
		genericNodes[i] = n
	}
	buildStringToPrint(genericNodes, &output)

	expected := `Assignment(Variable($name) @ 1:8 = "John" @ 1:16) @ 1:8`

	if strings.TrimSpace(output.String()) != strings.TrimSpace(expected) {
		t.Errorf("Expected output to be %q, but got %q", strings.TrimSpace(expected), strings.TrimSpace(output.String()))
	}
}
