package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestParseDNFReturnTypeWithTrailingIntersection(t *testing.T) {
	nodes := parseNoErrors(t, `<?php function openChannel(): null|(Readable&Writable) { return null; }`)
	if len(nodes) != 1 {
		t.Fatalf("expected one function, got %d nodes", len(nodes))
	}
	function, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("expected FunctionNode, got %T", nodes[0])
	}
	if function.ReturnType != "null|(Readable&Writable)" {
		t.Fatalf("expected DNF return type to be preserved, got %q", function.ReturnType)
	}
}

func TestParseDNFReturnTypeWithLeadingIntersection(t *testing.T) {
	nodes := parseNoErrors(t, `<?php function openStream(): (Readable&Seekable)|false { return false; }`)
	function, ok := nodes[0].(*ast.FunctionNode)
	if !ok {
		t.Fatalf("expected FunctionNode, got %T", nodes[0])
	}
	if function.ReturnType != "(Readable&Seekable)|false" {
		t.Fatalf("expected DNF return type to be preserved, got %q", function.ReturnType)
	}
}
