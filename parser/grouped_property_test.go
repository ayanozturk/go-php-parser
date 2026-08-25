package parser

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestParseGroupedPropertyDeclaration(t *testing.T) {
	nodes := parseNoErrors(t, `<?php class Coordinates { protected static ?int $x = null, $y, $z = 3; }`)
	classNode, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected ClassNode, got %T", nodes[0])
	}
	if len(classNode.Properties) != 3 {
		t.Fatalf("expected three properties, got %d", len(classNode.Properties))
	}
	for index, expectedName := range []string{"x", "y", "z"} {
		property, ok := classNode.Properties[index].(*ast.PropertyNode)
		if !ok {
			t.Fatalf("expected PropertyNode at %d, got %T", index, classNode.Properties[index])
		}
		if property.Name != expectedName || property.TypeHint != "?int" || property.Visibility != "protected" || !property.IsStatic {
			t.Fatalf("unexpected grouped property at %d: %#v", index, property)
		}
	}
	if classNode.Properties[0].(*ast.PropertyNode).DefaultValue == nil || classNode.Properties[1].(*ast.PropertyNode).DefaultValue != nil || classNode.Properties[2].(*ast.PropertyNode).DefaultValue == nil {
		t.Fatal("grouped property defaults were not preserved independently")
	}
}

func TestParseGroupedTraitPropertyDeclaration(t *testing.T) {
	parseNoErrors(t, `<?php trait TransitionState { private string $before, $after = ''; }`)
}
