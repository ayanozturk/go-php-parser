package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

var semanticSnapshotAllocationSink *SemanticSnapshot

func BenchmarkSemanticSnapshotConstructionAllocation(b *testing.B) {
	nodes, diags := syntax.ParseAST([]byte(`<?php
function AllocationProbeFunction(string $input): string {
    $one = trim($input);
    $two = trim($one);
    $three = trim($two);
    $four = trim($three);
    $five = trim($four);
    $six = trim($five);
    $seven = trim($six);
    $eight = trim($seven);
    return trim($eight);
}
`))
	if len(diags) > 0 {
		b.Fatalf("parse errors: %v", diags)
	}
	parsed := map[string][]ast.Node{"src/AllocationProbe.php": nodes}
	index := BuildProjectIndex(parsed)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot, err := NewSemanticSnapshotWithIndex(index, parsed, nil, nil)
		if err != nil {
			b.Fatal(err)
		}
		semanticSnapshotAllocationSink = snapshot
	}
}
