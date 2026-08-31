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
function allocation_probe(int $count, string $label, array $options): string {}
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

func TestSemanticSnapshotResolveFunctionIsDefensiveAndDeterministic(t *testing.T) {
	snapshot := resolverAllocationSnapshot(t)

	first, ok := snapshot.ResolveFunction("allocation_probe")
	if !ok {
		t.Fatal("expected allocation_probe function")
	}
	if got, want := len(first.Params), 3; got != want {
		t.Fatalf("allocation_probe parameter count = %d, want %d", got, want)
	}
	if first.Params[0].Name != "count" || first.Params[0].Type != "int" {
		t.Fatalf("unexpected allocation_probe parameter metadata: %#v", first.Params)
	}

	first.Name = "mutated"
	first.Params[0].Name = "callerOnly"
	first.Params[0].Type = "mutated"
	first.Params = append(first.Params, ResolvedParam{Name: "extra"})

	second, ok := snapshot.ResolveFunction("ALLOCATION_PROBE")
	if !ok {
		t.Fatal("expected allocation_probe function after caller mutation")
	}
	if second.Name != "allocation_probe" {
		t.Fatalf("function name leaked caller mutation: %q", second.Name)
	}
	if got, want := len(second.Params), 3; got != want {
		t.Fatalf("function parameter count leaked caller mutation: got %d, want %d", got, want)
	}
	if second.Params[0].Name != "count" || second.Params[0].Type != "int" {
		t.Fatalf("function parameter metadata leaked caller mutation: %#v", second.Params)
	}
}

func TestResolveFunctionViewReusesImmutableParams(t *testing.T) {
	snapshot := resolverAllocationSnapshot(t)

	first, ok := resolveFunctionView(snapshot, "allocation_probe")
	if !ok {
		t.Fatal("expected allocation_probe function view")
	}
	second, ok := resolveFunctionView(snapshot, "ALLOCATION_PROBE")
	if !ok {
		t.Fatal("expected allocation_probe function view on repeated lookup")
	}
	if got, want := len(first.Params), 3; got != want {
		t.Fatalf("function view parameter count = %d, want %d", got, want)
	}
	if &first.Params[0] != &second.Params[0] {
		t.Fatal("function view cloned immutable parameter storage between calls")
	}
	if first.Params[0].Name != "count" || first.Params[1].Name != "label" || first.Params[2].Name != "options" {
		t.Fatalf("unexpected function view parameters: %#v", first.Params)
	}
}

func consumeResolvedFunctionForAllocationTest(function ResolvedFunction) {
	resolverAllocationSink = function.Name
	if len(function.Params) > 0 {
		resolverAllocationSink = function.Params[len(function.Params)-1].Name
	}
}

func TestResolveFunctionViewReducesPerCallAllocations(t *testing.T) {
	snapshot := resolverAllocationSnapshot(t)

	// Warm both lookup paths before measuring. The setup allocation is
	// intentionally outside both AllocsPerRun bodies.
	public, ok := snapshot.ResolveFunction("allocation_probe")
	if !ok {
		t.Fatal("expected allocation_probe function")
	}
	consumeResolvedFunctionForAllocationTest(public)
	view, ok := resolveFunctionView(snapshot, "allocation_probe")
	if !ok {
		t.Fatal("expected allocation_probe function view")
	}
	consumeResolvedFunctionForAllocationTest(view)

	publicAllocs := testing.AllocsPerRun(100, func() {
		function, ok := snapshot.ResolveFunction("allocation_probe")
		if !ok {
			t.Fatal("public function lookup unexpectedly failed")
		}
		consumeResolvedFunctionForAllocationTest(function)
	})
	viewAllocs := testing.AllocsPerRun(100, func() {
		function, ok := resolveFunctionView(snapshot, "allocation_probe")
		if !ok {
			t.Fatal("function view lookup unexpectedly failed")
		}
		consumeResolvedFunctionForAllocationTest(function)
	})
	if viewAllocs >= publicAllocs {
		t.Fatalf("function view allocations = %v, public lookup allocations = %v; view should reuse immutable parameter storage", viewAllocs, publicAllocs)
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
