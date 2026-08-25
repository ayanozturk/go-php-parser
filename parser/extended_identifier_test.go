package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestParseRawExtendedByteClassIdentifier(t *testing.T) {
	name := string([]byte{0xa7})
	nodes := parseNoErrors(t, "<?php class "+name+" {}")
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if classNode.Name != name {
		t.Fatalf("expected raw extended-byte class name %q, got %q", name, classNode.Name)
	}
}

func TestParseRawExtendedByteVariableIdentifier(t *testing.T) {
	name := string([]byte{0xb6})
	parseNoErrors(t, "<?php $"+name+" = 1;")
}
