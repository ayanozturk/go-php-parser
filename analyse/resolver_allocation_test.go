package analyse

import (
	"reflect"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func resolverAllocationSnapshot(t *testing.T) *SemanticSnapshot {
	t.Helper()
	nodes := parsePHPForProjectIndex(t, `<?php
class AllocationProbe {
    public function Zulu(int $value): string {}
    protected function alpha(string $name): string {}
    private function middle(bool $enabled, array $values): string {}
}
`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{"src/AllocationProbe.php": nodes}, nil)
	if err != nil {
		t.Fatalf("build allocation fixture snapshot: %v", err)
	}
	return snapshot
}

func TestMethodsDeclaredByIsDefensiveAndDeterministic(t *testing.T) {
	snapshot := resolverAllocationSnapshot(t)

	first := snapshot.MethodsDeclaredBy("AllocationProbe")
	if got, want := resolvedMethodNames(first), []string{"alpha", "middle", "Zulu"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("declared method order = %v, want %v", got, want)
	}
	if len(first[0].Params) != 1 || first[0].Params[0].Type != "string" {
		t.Fatalf("unexpected first method metadata: %#v", first[0])
	}

	first[0].Name = "mutated"
	first[0].Params[0].Type = "mutated"
	first[0].Params = append(first[0].Params, ResolvedParam{Name: "callerOnly"})

	second := snapshot.MethodsDeclaredBy("allocationprobe")
	if got, want := resolvedMethodNames(second), []string{"alpha", "middle", "Zulu"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("declared method order changed after caller mutation: got %v, want %v", got, want)
	}
	if len(second[0].Params) != 1 || second[0].Params[0].Type != "string" {
		t.Fatalf("declared method metadata leaked caller mutation: %#v", second[0])
	}
	if !reflect.DeepEqual(second, snapshot.MethodsDeclaredBy("AllocationProbe")) {
		t.Fatalf("repeated declared-method query was not deterministic:\nfirst %#v\nsecond %#v", second, snapshot.MethodsDeclaredBy("AllocationProbe"))
	}
}

func TestRangeMethodsDeclaredByVisitsStableOrderedMethodsWithoutParamClones(t *testing.T) {
	snapshot := resolverAllocationSnapshot(t)

	var firstNames []string
	var firstParam *ResolvedParam
	rangeMethodsDeclaredBy(snapshot, "AllocationProbe", func(method ResolvedMethod) bool {
		firstNames = append(firstNames, method.Name)
		if method.Name == "alpha" {
			firstParam = &method.Params[0]
		}
		return true
	})
	if want := []string{"alpha", "middle", "Zulu"}; !reflect.DeepEqual(firstNames, want) {
		t.Fatalf("allocation-light method order = %v, want %v", firstNames, want)
	}

	var secondParam *ResolvedParam
	rangeMethodsDeclaredBy(snapshot, "allocationprobe", func(method ResolvedMethod) bool {
		if method.Name == "alpha" {
			secondParam = &method.Params[0]
		}
		return true
	})
	if firstParam == nil || secondParam == nil {
		t.Fatal("allocation-light traversal did not visit alpha parameter metadata")
	}
	if firstParam != secondParam {
		t.Fatal("allocation-light traversal cloned parameter storage between calls")
	}
}

var resolverAllocationSink string

func consumeResolvedMethodForAllocationTest(method ResolvedMethod) bool {
	resolverAllocationSink = method.Name
	return true
}

func TestRangeMethodsDeclaredByReducesPerCallAllocations(t *testing.T) {
	snapshot := resolverAllocationSnapshot(t)

	// Warm the immutable member view and the public query before measuring. The
	// setup allocation is intentionally outside both AllocsPerRun bodies.
	_ = snapshot.MethodsDeclaredBy("AllocationProbe")
	rangeMethodsDeclaredBy(snapshot, "AllocationProbe", consumeResolvedMethodForAllocationTest)

	publicAllocs := testing.AllocsPerRun(100, func() {
		methods := snapshot.MethodsDeclaredBy("AllocationProbe")
		if len(methods) != 0 {
			resolverAllocationSink = methods[len(methods)-1].Name
		}
	})
	traversalAllocs := testing.AllocsPerRun(100, func() {
		rangeMethodsDeclaredBy(snapshot, "AllocationProbe", consumeResolvedMethodForAllocationTest)
	})
	if traversalAllocs >= publicAllocs {
		t.Fatalf("allocation-light traversal allocations = %v, public query allocations = %v; traversal should avoid per-call collection and parameter cloning", traversalAllocs, publicAllocs)
	}
}

func resolvedMethodNames(methods []ResolvedMethod) []string {
	names := make([]string, 0, len(methods))
	for _, method := range methods {
		names = append(names, method.Name)
	}
	return names
}
